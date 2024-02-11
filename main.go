package main

import (
	"errors"
	"fmt"
	"log"
	"reflect"
)

type Service struct{}

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

			if in.Kind() == reflect.Slice {
				srv.namespaces[service].methods[rt.Method(i).Name].in[j] = in
				srv.namespaces[service].methods[rt.Method(i).Name].inString[j] = fmt.Sprintf(
					"[]%s",
					in.Elem().String(),
				)
			} else {
				srv.namespaces[service].methods[rt.Method(i).Name].in[j] = in
				srv.namespaces[service].methods[rt.Method(i).Name].inString[j] = in.Kind().String()
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

func (srv *Server) Call(namespace string, method string, args []interface{}) {

	target := srv.namespaces[namespace].methods[method]
	in := make([]reflect.Value, len(target.in))

	if len(target.in) == 0 {
	} else {
		// TODO: подумать, что делать с вариантом args ...type
		if len(args) != len(target.in) {
			return
		}

		for i := 0; i < len(args); i++ {
			v := reflect.New(target.in[i]).Elem()

			if target.in[i].Kind() == reflect.Slice {
				// NEXT: Если в аргументах передали Slice нужно научится создавать свой собственный слайс с заданным типом
				// []interface{ []int{1,2} }  -> нужно определить, что в первом элементе находится Slice и тип int,
				// после этого создать reflect.Value от Slice и заполнить его значениями reflect.Value от элементов Slice-a
				va := reflect.MakeSlice(v.Type(), len(args[i]), len(args[i]))
				for j := 0; j < len(args[i]); j++ {
					item := va.Index(j)
					item.Set()
				}
				println(v.Kind().String() + " = " + va.Kind().String())
			} else {
				va := reflect.ValueOf(args[i])
				v.Set(va)
			}

			if !v.CanSet() {
				println("cant set")
			}
			in[i] = v
		}
	}
	val := target.value.Call(in)
	fmt.Printf("%+v\n", val[0].Interface())
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

func main() {
	println()
	tmp := &Service{}

	server := NewServer()

	_ = server.Register("service", tmp)
	server.Call("service", "Method1", []interface{}{})

	server.Call("service", "Method2", []interface{}{1, 2})
	server.Call("service", "Method3", []interface{}{
		[]interface{}{1, 4},
	})
}
