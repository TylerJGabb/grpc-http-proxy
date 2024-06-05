package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/TylerJGabb/grpc-http-proxy/pkg/tgsbpb"
)

func debugGoroutineCount(wg *sync.WaitGroup) {
	go func() {
		ch := make(chan struct{})
		go func() {
			wg.Wait()
			close(ch)
		}()
		select {
		case <-ch:
			fmt.Print("goroutine leak check passed\n")
		case <-time.After(time.Second * 5):
			fmt.Println("!!GOROUTINE LEAK!! timeout waiting for all responses to be sent")
		}
	}()
}

func (s *exampleServer) BidirectionalStreamString(
	stream tgsbpb.TylerSandboxService_BidirectionalStreamStringServer,
) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()
	in := make(chan *tgsbpb.BidirectionalStreamStringRequest)
	errs := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(2)
	defer debugGoroutineCount(wg)
	go func() {
		defer wg.Done()
		defer fmt.Println("receiver done")
		for {
			msg, err := stream.Recv()
			if err != nil {
				fmt.Printf("receive error: %s\n", err.Error())
				errs <- err
				return
			}
			select {
			case in <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		defer fmt.Println("sender done")
		for {
			select {
			case <-ctx.Done():
				return
			case req := <-in:
				if req.Close {
					fmt.Println("closing")
					cancel()
					return
				}
				if req.RespondWithError {
					fmt.Println("intentionally returned error")
					errs <- fmt.Errorf("intentionally returned error")
					return
				}
				send := &tgsbpb.BidirectionalStreamStringResponse{Value: req.Value}
				if err := stream.Send(send); err != nil {
					fmt.Printf("send error: %s\n", err.Error())
					errs <- err
					return
				}
			}
		}
	}()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return nil
	}
}
