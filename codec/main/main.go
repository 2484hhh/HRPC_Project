package main

import (
	"context"
	"log"
	"net"
	"net/http"
	hrpc "rpcproject/codec"
	"rpcproject/codec/client"
	"rpcproject/codec/registry"
	"sync"
	"time"
)

type Foo int
type Foo2 int

type Args struct{ Num1, Num2 int }

func (f Foo) Sum(args Args, reply *int) error {
	*reply = args.Num1 + args.Num2
	return nil
}

func (f Foo2) Sub(args Args, reply *int) error {
	*reply = args.Num2 - args.Num1
	return nil
}

func (f Foo) Sleep(args Args, reply *int) error {
	time.Sleep(time.Second * time.Duration(args.Num1))
	*reply = args.Num1 + args.Num2
	return nil
}

func startRegistry(wg *sync.WaitGroup) {
	l, _ := net.Listen("tcp", ":9999")
	registry.HandleHTTP()
	wg.Done()
	http.Serve(l, nil)
}
func startServer(registryAddr string, wg *sync.WaitGroup) {
	var foo Foo
	var foo2 Foo2
	l, _ := net.Listen("tcp", ":0")
	server := hrpc.NewServer()
	server.Register(&foo)
	server.Register(&foo2)
	registry.Heartbeat(registryAddr, "tcp@"+l.Addr().String(), 0)
	wg.Done()
	server.Accept(l)
}

func foo(x *client.XClient, ctx context.Context, typ, serviceMethod string, args *Args) {
	var reply int
	var err error
	switch typ {
	case "call":
		err = x.Call(ctx, serviceMethod, args, &reply)
	case "broadcast":
		err = x.Broadcast(ctx, serviceMethod, args, &reply)
	}
	if err != nil {
		log.Printf("%s %s error: %v", typ, serviceMethod, err)
	} else {
		switch serviceMethod {
		case "Foo.Sum":
			log.Printf("%s %s success: %d + %d =%d", typ, serviceMethod, args.Num1, args.Num2, reply)
		case "Foo2.Sub":
			log.Printf("%s %s success: %d - %d =%d", typ, serviceMethod, args.Num2, args.Num1, reply)
		case "Foo.Sleep":
			log.Printf("%s %s success: %d + %d =%d", typ, serviceMethod, args.Num1, args.Num2, reply)
		}
	}

}

func call(registry string) {
	d := client.NewHRegistryDiscovery(registry, 0)
	c := client.NewXClient(d, client.RoundRobinSelect, nil)
	defer c.Close()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(c, context.Background(), "call", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			foo(c, context.Background(), "call", "Foo2.Sub", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func broadcast(registry string) {
	d := client.NewHRegistryDiscovery(registry, 0)
	c := client.NewXClient(d, client.RoundRobinSelect, nil)
	defer c.Close()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			foo(c, context.Background(), "broadcast", "Foo.Sum", &Args{Num1: i, Num2: i * i})
			foo(c, context.Background(), "broadcast", "Foo2.Sub", &Args{Num1: i, Num2: i * i})

			ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
			foo(c, ctx, "broadcast", "Foo.Sleep", &Args{Num1: i, Num2: i * i})
		}(i)
	}
	wg.Wait()
}

func main() {
	log.SetFlags(0)
	registryAddr := "http://localhost:9999/_hrpc_/registry"
	var wg sync.WaitGroup
	wg.Add(1)
	go startRegistry(&wg)
	wg.Wait()

	time.Sleep(time.Second)
	wg.Add(2)
	go startServer(registryAddr, &wg)
	go startServer(registryAddr, &wg)
	wg.Wait()

	time.Sleep(time.Second)
	call(registryAddr)
	broadcast(registryAddr)
}
