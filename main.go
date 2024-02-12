package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
)

type Service struct{}

const (
	JsonRpcParseError     = -32700
	JsonRpcInvalidRequest = -32600
	JsonRpcMethodNotFound = -32601
	JsonRpcInvalidaParams = -32602
	JsonRpcInternalError  = -32603
)

type Server struct {
	namespaces map[string]Namespace
}

type Namespace struct {
	methods map[string]Method
}

type Method struct {
	value     reflect.Value
	in        []reflect.Type
	out       []reflect.Type
	inString  []string
	outString []string
}

func NewServer() *Server {
	return &Server{
		namespaces: make(map[string]Namespace),
	}
}

func (srv *Server) Register(service string, obj any) error {
	// check if obj is struct instance

	r := reflect.ValueOf(obj)

	// check if kind of obj == reflect.Struct
	if r.Kind() == reflect.Ptr {
		log.Println("it's a pointer")
		if r.Elem().Kind() == reflect.Struct {
			log.Println("it's a struct")
		} else {
			return errors.New("ptr on struct expected")
		}
	} else {
		return errors.New("pointer expected")
	}

	// list of methods
	rt := reflect.TypeOf(obj)
	srv.namespaces[service] = Namespace{
		methods: make(map[string]Method),
	}

	for i := 0; i < rt.NumMethod(); i++ {
		method := r.Method(i)

		srv.namespaces[service].methods[rt.Method(i).Name] = Method{
			value:     method,
			in:        make([]reflect.Type, method.Type().NumIn()),
			out:       make([]reflect.Type, method.Type().NumOut()),
			inString:  make([]string, method.Type().NumIn()),
			outString: make([]string, method.Type().NumOut()),
		}
		for j := 0; j < method.Type().NumIn(); j++ {
			in := method.Type().In(j)

			switch in.Kind() {
			case reflect.Slice:
				srv.namespaces[service].methods[rt.Method(i).Name].in[j] = in
				srv.namespaces[service].methods[rt.Method(i).Name].inString[j] = fmt.Sprintf(
					"[]%s",
					in.Elem().String(),
				)
			case reflect.Struct:
				srv.namespaces[service].methods[rt.Method(i).Name].in[j] = in
				srv.namespaces[service].methods[rt.Method(i).Name].inString[j] = in.Kind().String()
			default:
				srv.namespaces[service].methods[rt.Method(i).Name].in[j] = in
				srv.namespaces[service].methods[rt.Method(i).Name].inString[j] = in.String()
			}
		}

		for j := 0; j < method.Type().NumOut(); j++ {
			out := method.Type().Out(j)

			switch out.Kind() {
			case reflect.Interface:
				srv.namespaces[service].methods[rt.Method(i).Name].out[j] = out
				srv.namespaces[service].methods[rt.Method(i).Name].outString[j] = out.String()
			case reflect.Slice:
				srv.namespaces[service].methods[rt.Method(i).Name].out[j] = out
				srv.namespaces[service].methods[rt.Method(i).Name].outString[j] = fmt.Sprintf(
					"[]%s",
					out.String(),
				)
			default:
				srv.namespaces[service].methods[rt.Method(i).Name].out[j] = out
				srv.namespaces[service].methods[rt.Method(i).Name].outString[j] = out.Kind().
					String()
			}

		}

	}

	return nil
}

func (srv *Server) decoderType(source, destination reflect.Value) (reflect.Value, error) {

	var result reflect.Value

	if source.Type() == destination.Type() {
		return source, nil
	}

	if source.Kind() == reflect.Float64 && destination.Kind() == reflect.Int {
		return source.Convert(destination.Type()), nil
	} else {
		return source, nil
	}

	return result, nil

}

func (srv *Server) call(namespace string, method string, args []interface{}) (interface{}, error) {

	target := srv.namespaces[namespace].methods[method]
	in := make([]reflect.Value, len(target.in))
	if len(target.in) == 0 {
	} else {
		if len(args) != len(target.in) {
			return nil, fmt.Errorf("length of agrs is not enough")
		}

		for i := 0; i < len(args); i++ {
			v := reflect.New(target.in[i]).Elem()
			va := reflect.ValueOf(args[i])

			switch target.in[i].Kind() {
			case reflect.Struct:
				data, err := json.Marshal(args[i])
				if err != nil {
					println(err.Error())
					return nil, nil
				}

				structVO := reflect.New(target.in[i]).Interface()

				err = json.Unmarshal(data, structVO)
				if err != nil {
					println(err.Error())
					return nil, nil
				}
				va = reflect.ValueOf(structVO).Elem()
				v.Set(va)

			case reflect.Slice:
				if va.Kind() != reflect.Slice {
					return nil, fmt.Errorf("argument should be a slice, %s given", va.Kind().String())
				}
				v = reflect.MakeSlice(target.in[i], va.Len(), va.Cap())

				// получаем Slice от args[i]
				for j := 0; j < va.Len(); j++ {
					vCurrent := reflect.New(v.Index(j).Type()).Elem()

					if !vCurrent.CanSet() {
						continue
					}

					convertedVO, err := srv.decoderType(va.Index(j).Elem(), vCurrent)
					if err != nil {
						println(err.Error())
						return nil, err
					}

					vCurrent.Set(convertedVO)
					//vCurrent.Set(va.Index(j).Elem())
					if !v.Index(j).CanSet() {
						println("cant set to a slice")
					}
					v.Index(j).Set(vCurrent)
				}
			default:
				if !v.CanSet() {
					println("cant set")
				}
				convertedVO, err := srv.decoderType(va, v)
				if err != nil {
					println(err.Error())
					return nil, err
				}
				v.Set(convertedVO)
			}

			in[i] = v
		}
	}

	val := target.value.Call(in)
	fmt.Printf("%+v\n", val[0].Interface())
	return val, nil
}

type JsonRpcRequest struct {
	JsonRpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	Id      *string       `json:"id,omitempty"`
}

type JsonRpcResponse struct {
	JsonRpc string               `json:"jsonrpc"`
	Id      *string              `json:"id"`
	Error   JsonRpcResponseError `json:"error,omitempty"`
}

type JsonRpcResponseError struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

func (srv *Server) Handler(data []byte) {

	var request JsonRpcRequest

	err := json.Unmarshal(data, &request)
	if err != nil {
		println(err.Error())
		return
	}

	if request.Id == nil || len(*request.Id) == 0 {
		println("id not set")
		return
	}

	path := strings.Split(request.Method, ".")
	if len(path) != 2 {
		return
	}

	namespace, method := path[0], path[1]
	if _, ok := srv.namespaces[namespace]; !ok {
		println("namespace not found")
		return
	}

	if _, ok := srv.namespaces[namespace].methods[method]; !ok {
		println("method not found")
		return
	}

	srv.call(namespace, method, request.Params)

}

func (srv *Service) Method1() (string, error) {
	return "result of method1", nil
}

func (srv *Service) Method2(a int, b int) (int, error) {
	return a + b, nil
}

func (srv *Service) Method3(a []int) (int, error) {
	if len(a) < 2 {
		return 0, errors.New("wrong arguments number")
	}
	result := 0
	for _, item := range a {
		result += item
	}

	return result, nil
}

func (srv *Service) Method4(a []float64) (float64, error) {
	if len(a) < 2 {
		return 0, errors.New("wrong arguments number")
	}
	result := 0.0
	for _, item := range a {
		result += item
	}

	return result, nil
}

type Request struct {
	Name     string `json:"name"`
	Lastname string `json:"lastname"`
	Phones   []int  `json:"phones"`
}

func (srv *Service) Method5(obj Request) (string, error) {
	fmt.Printf("%+v\n", obj)
	return "result of method5", nil
}

func main() {
	println()
	tmp := &Service{}

	server := NewServer()

	_ = server.Register("service", tmp)
	/*
		server.Call("service", "Method1", []interface{}{})

		server.Call("service", "Method2", []interface{}{1, 2})
		server.Call("service", "Method3", []interface{}{
			[]interface{}{2, 4},
		})
		server.Call("service", "Method4", []interface{}{
			[]interface{}{2.0, 4.0},
		})
	*/

	server.Handler([]byte(`{"jsonrpc": "2.0", "method": "service.Method2", "params": [1,10], "id":"1"}`))
	server.Handler([]byte(`{"jsonrpc": "2.0", "method": "service.Method3", "params": [[1,2,7]], "id":"1"}`))
	server.Handler([]byte(`{"jsonrpc": "2.0", "method": "service.Method4", "params": [[1,2]], "id":"1"}`))
	server.Handler([]byte(`{"jsonrpc": "2.0", "method": "service.Method5", "params": [{"name":"maxim", "phones":[123,456]}], "id":"1"}`))
}
