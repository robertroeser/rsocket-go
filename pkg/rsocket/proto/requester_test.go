package proto

import (
	"context"
	"flag"
	"os"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/zap"

	"github.com/flier/rsocket-go/pkg/rsocket/frame"
)

var logger *zap.Logger

func TestMain(m *testing.M) {
	flag.Parse()

	if testing.Verbose() {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}

	defer logger.Sync()

	os.Exit(m.Run())
}

func TestRequestStreamComplete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame)
	initReqs := 16

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), uint(initReqs))

		// RQ -> RS: REQUEST_STREAM
		// RS -> RQ: PAYLOAD*
		// RS -> RQ: COMPLETE
		Convey("When request stream for payloads", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()

				Convey("Then request should be sent", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestStream)
					So(f.Flags(), ShouldEqual, frame.FlagMetadata)

					requestFrame := f.(*frame.RequestStreamFrame)

					So(requestFrame, ShouldNotBeNil)
					So(requestFrame.InitialRequests, ShouldEqual, initReqs)
					So(string(requestFrame.Data), ShouldEqual, "hello")
					So(string(requestFrame.Metadata), ShouldEqual, "world")

					Convey("Then send payload", func() {
						payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))

						So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)

						payloadFrame = buildPayloadFrame(f.StreamID(), true, Text("bar"))

						So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)
					})
				})
			}()

			responses, err := requester.RequestStream(ctx, Text("hello").WithMetadata([]byte("world")))

			So(err, ShouldBeNil)

			Convey("Then payload stream should be ready", func() {
				result := func() *Result {
					select {
					case <-ctx.Done():
						return &Result{nil, ctx.Err()}

					case result, ok := <-responses:
						if !ok {
							return nil
						}

						return result
					}
				}

				So(result().Payload, ShouldResemble, Text("foo"))
				So(result().Payload, ShouldResemble, Text("bar"))
				So(result(), ShouldBeNil)
			})
		})
	})
}

func TestRequestStreamWithError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame)
	initReqs := 16

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), uint(initReqs))

		// RQ -> RS: REQUEST_STREAM
		// RS -> RQ: PAYLOAD*
		// RS -> RQ: ERROR[APPLICATION_ERROR|REJECTED|CANCELED|INVALID]
		Convey("When request stream for payloads with error", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()

				Convey("Then request should be sent", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestStream)
					So(f.Flags(), ShouldEqual, frame.FlagMetadata)

					requestFrame := f.(*frame.RequestStreamFrame)

					So(requestFrame, ShouldNotBeNil)
					So(requestFrame.InitialRequests, ShouldEqual, initReqs)
					So(string(requestFrame.Data), ShouldEqual, "hello")
					So(string(requestFrame.Metadata), ShouldEqual, "world")

					Convey("Then send payload with error", func() {
						payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))
						So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)

						payloadFrame = buildPayloadFrame(f.StreamID(), false, Text("bar"))
						So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)

						errorFrame := frame.NewErrorFrame(f.StreamID(), frame.ErrApplicationError, "for test")
						So(requester.(*rSocketRequester).handleFrame(ctx, errorFrame), ShouldBeNil)
					})
				})
			}()

			responses, err := requester.RequestStream(ctx, Text("hello").WithMetadata([]byte("world")))

			So(err, ShouldBeNil)

			Convey("Then payload stream should be ready", func() {
				result := func() *Result {
					select {
					case <-ctx.Done():
						return &Result{nil, ctx.Err()}

					case result, ok := <-responses:
						if !ok {
							return nil
						}

						return result
					}
				}

				So(result().Payload, ShouldResemble, Text("foo"))
				So(result().Payload, ShouldResemble, Text("bar"))
				So(result().Err, ShouldResemble, frame.ErrApplicationError.WithMessage("for test"))
				So(result(), ShouldBeNil)
			})
		})
	})
}

func TestRequestStreamCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame)
	initReqs := 16

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), uint(initReqs))

		// RQ -> RS: REQUEST_STREAM
		// RS -> RQ: PAYLOAD*
		// RQ -> RS: CANCEL
		Convey("When request stream for payloads with cancel", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()

				Convey("Then request should be sent", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestStream)
					So(f.Flags(), ShouldEqual, frame.FlagMetadata)

					requestFrame := f.(*frame.RequestStreamFrame)

					So(requestFrame, ShouldNotBeNil)
					So(requestFrame.InitialRequests, ShouldEqual, initReqs)
					So(string(requestFrame.Data), ShouldEqual, "hello")
					So(string(requestFrame.Metadata), ShouldEqual, "world")

					Convey("Then send payload with error", func() {
						payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))
						So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)

						payloadFrame = buildPayloadFrame(f.StreamID(), false, Text("bar"))
						So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)

						cancelFrame := frame.NewCancelFrame(f.StreamID())
						So(requester.(*rSocketRequester).handleFrame(ctx, cancelFrame), ShouldBeNil)
					})
				})
			}()

			responses, err := requester.RequestStream(ctx, Text("hello").WithMetadata([]byte("world")))

			So(err, ShouldBeNil)

			Convey("Then payload stream should be ready", func() {
				result := func() *Result {
					select {
					case <-ctx.Done():
						return &Result{nil, ctx.Err()}

					case result, ok := <-responses:
						if !ok {
							return nil
						}

						return result
					}
				}

				So(result().Payload, ShouldResemble, Text("foo"))
				So(result().Payload, ShouldResemble, Text("bar"))
				So(result().Err, ShouldEqual, context.Canceled)
				So(result(), ShouldBeNil)
			})
		})
	})
}

func TestRequestChannelCompleteFromRequesterAndResponder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame, 16)
	initReqs := 16

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), uint(initReqs))

		// RQ -> RS: REQUEST_CHANNEL
		// RQ -> RS: PAYLOAD*
		// RQ -> RS: COMPLETE
		//
		// intermixed with
		//
		// RS -> RQ: PAYLOAD*
		// RS -> RQ: COMPLETE
		Convey("When COMPLETE from Requester and Responder", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()
				Convey("Then request should be ready", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestChannel)
					So(f.Flags(), ShouldEqual, 0)

					Convey("Then payload should be embedded", func() {
						requestFrame := f.(*frame.RequestChannelFrame)

						So(requestFrame, ShouldNotBeNil)
						So(requestFrame.InitialRequests, ShouldEqual, initReqs)
						So(string(requestFrame.Data), ShouldEqual, "hello")

						Convey("Then send requestN back to requester", func() {
							requestNFrame := frame.NewRequestNFrame(f.StreamID(), uint32(initReqs))

							So(requester.(*rSocketRequester).handleFrame(ctx, requestNFrame), ShouldBeNil)

							Convey("Then payload should be sent", func() {
								f := <-frameSender

								So(f, ShouldNotBeNil)
								So(f.StreamID(), ShouldEqual, 1)
								So(f.Type(), ShouldEqual, frame.TypePayload)
								So(f.Flags(), ShouldEqual, frame.FlagNext)

								payloadFrame := f.(*frame.PayloadFrame)

								So(payloadFrame, ShouldNotBeNil)
								So(string(payloadFrame.Data), ShouldEqual, "world")

								Convey("Then requests stream should be complete", func() {
									f := <-frameSender

									So(f, ShouldNotBeNil)
									So(f.StreamID(), ShouldEqual, 1)
									So(f.Type(), ShouldEqual, frame.TypePayload)
									So(f.Flags(), ShouldEqual, frame.FlagComplete)

									payloadFrame := f.(*frame.PayloadFrame)

									So(payloadFrame, ShouldNotBeNil)
									So(payloadFrame.Data, ShouldBeNil)

									Convey("Then send payload", func() {
										payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("foo"))

										So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)

										payloadFrame = buildPayloadFrame(f.StreamID(), true, Text("bar"))

										So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)
									})
								})
							})
						})
					})
				})
			}()

			requests := make(chan *Result, 128)

			Convey("When payloads be ready before send request", func() {
				send := func(payload *Payload) error {
					select {
					case <-ctx.Done():
						return ctx.Err()

					case requests <- &Result{payload, nil}:
						return nil
					}
				}

				So(send(Text("hello")), ShouldBeNil)
				So(send(Text("world")), ShouldBeNil)
				close(requests)

				responses, err := requester.RequestChannel(ctx, requests)

				So(err, ShouldBeNil)

				Convey("Then payload stream should be ready", func() {
					result := func() *Result {
						select {
						case <-ctx.Done():
							return &Result{nil, ctx.Err()}

						case result, ok := <-responses:
							if !ok {
								return nil
							}

							return result
						}
					}

					So(result().Payload, ShouldResemble, Text("foo"))
					So(result().Payload, ShouldResemble, Text("bar"))
					So(result(), ShouldBeNil)
				})
			})
		})
	})
}

func TestRequestChannelErrorFromRequesterAndResponderTerminates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame, 16)
	initReqs := 16

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), uint(initReqs))
		// RQ -> RS: REQUEST_CHANNEL
		// RQ -> RS: PAYLOAD*
		// RQ -> RS: ERROR[APPLICATION_ERROR]
		//
		// intermixed with
		//
		// RS -> RQ: PAYLOAD*
		Convey("When Error from Requester, Responder terminates", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()

				Convey("Then channel request should be ready", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestChannel)
					So(f.Flags(), ShouldEqual, 0)

					Convey("Then request payload should be nil", func() {
						requestFrame := f.(*frame.RequestChannelFrame)

						So(requestFrame, ShouldNotBeNil)
						So(requestFrame.InitialRequests, ShouldEqual, initReqs)
						So(requestFrame.Data, ShouldBeNil)

						Convey("Then send requestN back to requester", func() {
							requestNFrame := frame.NewRequestNFrame(f.StreamID(), uint32(initReqs))

							So(requester.(*rSocketRequester).handleFrame(ctx, requestNFrame), ShouldBeNil)

							Convey("Then payload should be sent", func() {
								f := <-frameSender

								So(f, ShouldNotBeNil)
								So(f.StreamID(), ShouldEqual, 1)
								So(f.Type(), ShouldEqual, frame.TypePayload)
								So(f.Flags(), ShouldEqual, frame.FlagNext)

								payloadFrame := f.(*frame.PayloadFrame)

								So(payloadFrame, ShouldNotBeNil)
								So(string(payloadFrame.Data), ShouldEqual, "hello")

								Convey("Then send response", func() {
									payloadFrame := buildPayloadFrame(f.StreamID(), false, Text("world"))

									So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)

									Convey("Then error should be sent", func() {
										f := <-frameSender

										So(f, ShouldNotBeNil)
										So(f.StreamID(), ShouldEqual, 1)
										So(f.Type(), ShouldEqual, frame.TypeError)
										So(f.Flags(), ShouldEqual, 0)

										errorFrame := f.(*frame.ErrorFrame)

										So(errorFrame, ShouldNotBeNil)
										So(errorFrame.Code, ShouldEqual, frame.ErrApplicationError)
										So(errorFrame.Data, ShouldEqual, "for test")
									})
								})
							})
						})
					})
				})
			}()

			requests := make(chan *Result, 128)

			Convey("Then send request immediately", func() {
				responses, err := requester.RequestChannel(ctx, requests)

				So(err, ShouldBeNil)
				Convey("When payloads sent after request", func() {
					send := func(result *Result) error {
						select {
						case <-ctx.Done():
							return ctx.Err()

						case requests <- result:
							return nil
						}
					}

					So(send(Ok(Text("hello"))), ShouldBeNil)

					Convey("Then payload stream should be ready", func() {
						result := func() *Result {
							select {
							case <-ctx.Done():
								return &Result{nil, ctx.Err()}

							case result, ok := <-responses:
								if !ok {
									return nil
								}

								return result
							}
						}

						So(result().Payload, ShouldResemble, Text("world"))

						Convey("Then send error", func() {
							So(send(Err(frame.ErrApplicationError.WithMessage("for test"))), ShouldBeNil)
							close(requests)

							Convey("Then the reciever should be close", func() {
								So(result(), ShouldBeNil)
							})
						})
					})
				})
			})
		})
	})
}

func TestRequestChannelErrorFromRequesterAndResponderAlreadyCompleted(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame, 16)
	initReqs := 16

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), uint(initReqs))
		// RQ -> RS: REQUEST_CHANNEL
		// RQ -> RS: PAYLOAD*
		// RQ -> RS: ERROR[APPLICATION_ERROR]
		//
		// intermixed with
		//
		// RS -> RQ: PAYLOAD*
		// RS -> RQ: COMPLETE
		Convey("When Error from Requester, Responder already Completed", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()

				Convey("Then channel request should be ready", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestChannel)
					So(f.Flags(), ShouldEqual, 0)

					Convey("Then request payload should be nil", func() {
						requestFrame := f.(*frame.RequestChannelFrame)

						So(requestFrame, ShouldNotBeNil)
						So(requestFrame.InitialRequests, ShouldEqual, initReqs)
						So(requestFrame.Data, ShouldBeNil)

						Convey("Then send requestN back to requester", func() {
							requestNFrame := frame.NewRequestNFrame(f.StreamID(), uint32(initReqs))

							So(requester.(*rSocketRequester).handleFrame(ctx, requestNFrame), ShouldBeNil)

							Convey("Then payload should be sent", func() {
								f := <-frameSender

								So(f, ShouldNotBeNil)
								So(f.StreamID(), ShouldEqual, 1)
								So(f.Type(), ShouldEqual, frame.TypePayload)
								So(f.Flags(), ShouldEqual, frame.FlagNext)

								payloadFrame := f.(*frame.PayloadFrame)

								So(payloadFrame, ShouldNotBeNil)
								So(string(payloadFrame.Data), ShouldEqual, "hello")

								Convey("Then send response", func() {
									payloadFrame := buildPayloadFrame(f.StreamID(), true, Text("world"))

									So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)

									Convey("Then error should be sent", func() {
										f := <-frameSender

										So(f, ShouldNotBeNil)
										So(f.StreamID(), ShouldEqual, 1)
										So(f.Type(), ShouldEqual, frame.TypeError)
										So(f.Flags(), ShouldEqual, 0)

										errorFrame := f.(*frame.ErrorFrame)

										So(errorFrame, ShouldNotBeNil)
										So(errorFrame.Code, ShouldEqual, frame.ErrApplicationError)
										So(string(errorFrame.Data), ShouldEqual, "for test")

									})
								})
							})
						})
					})
				})
			}()

			requests := make(chan *Result, 128)

			Convey("Then send request immediately", func() {
				responses, err := requester.RequestChannel(ctx, requests)

				So(err, ShouldBeNil)
				Convey("When payloads sent after request", func() {
					send := func(result *Result) error {
						select {
						case <-ctx.Done():
							return ctx.Err()

						case requests <- result:
							return nil
						}
					}

					So(send(Ok(Text("hello"))), ShouldBeNil)

					Convey("Then payload stream should be ready", func() {
						result := func() *Result {
							select {
							case <-ctx.Done():
								return &Result{nil, ctx.Err()}

							case result, ok := <-responses:
								if !ok {
									return nil
								}

								return result
							}
						}

						So(result().Payload, ShouldResemble, Text("world"))

						Convey("Then send error", func() {
							So(send(Err(frame.ErrApplicationError.WithMessage("for test"))), ShouldBeNil)
							close(requests)

							Convey("Then the reciever should be close", func() {
								So(result(), ShouldBeNil)
							})
						})
					})
				})
			})
		})
	})
}

func TestRequestResponseComplete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame)

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), 16)

		// RQ -> RS: REQUEST_RESPONSE
		// RS -> RQ: PAYLOAD with COMPLETE
		Convey("When request for response", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()

				Convey("The request should be sent", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestResponse)
					So(f.Flags(), ShouldEqual, frame.FlagMetadata)

					requestFrame := f.(*frame.RequestResponseFrame)

					So(requestFrame, ShouldNotBeNil)
					So(string(requestFrame.Data), ShouldEqual, "hello")
					So(string(requestFrame.Metadata), ShouldEqual, "world")

					Convey("Then send payload", func() {
						payloadFrame := buildPayloadFrame(f.StreamID(), true, Text("hello world"))

						So(requester.(*rSocketRequester).handleFrame(ctx, payloadFrame), ShouldBeNil)
					})
				})
			}()

			payload, err := requester.RequestResponse(ctx, Text("hello").WithMetadata([]byte("world")))

			So(err, ShouldBeNil)
			So(payload.Text(), ShouldEqual, "hello world")
		})
	})
}

func TestRequestResponseWithError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame)

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), 16)

		// RQ -> RS: REQUEST_RESPONSE
		// RS -> RQ: ERROR[APPLICATION_ERROR|REJECTED|CANCELED|INVALID]
		Convey("When request for response with error", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()

				Convey("The request should be sent", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestResponse)
					So(f.Flags(), ShouldEqual, 0)

					requestFrame := f.(*frame.RequestResponseFrame)

					So(requestFrame, ShouldNotBeNil)
					So(string(requestFrame.Data), ShouldEqual, "hello")

					Convey("Then send error", func() {
						errorFrame := frame.NewErrorFrame(f.StreamID(), frame.ErrApplicationError, "for test")

						So(requester.(*rSocketRequester).handleFrame(ctx, errorFrame), ShouldBeNil)
					})
				})
			}()

			payload, err := requester.RequestResponse(ctx, Text("hello"))

			So(payload, ShouldBeNil)
			So(err.Error(), ShouldEqual, "ERROR[APPLICATION_ERROR] for test")
			So(err.(*Error), ShouldResemble, frame.ErrApplicationError.WithMessage("for test"))
		})
	})
}

func TestRequestResponseCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame)

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), 16)

		// RQ -> RS: REQUEST_RESPONSE
		// RQ -> RS: CANCEL
		Convey("When request for response with cancel", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()

				Convey("The request should be sent", t, func() {
					f := <-frameSender

					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestResponse)
					So(f.Flags(), ShouldEqual, 0)

					requestFrame := f.(*frame.RequestResponseFrame)

					So(requestFrame, ShouldNotBeNil)
					So(string(requestFrame.Data), ShouldEqual, "hello")

					Convey("Then send cancel", func() {
						cancelFrame := frame.NewCancelFrame(f.StreamID())

						So(requester.(*rSocketRequester).handleFrame(ctx, cancelFrame), ShouldBeNil)
					})
				})
			}()

			payload, err := requester.RequestResponse(ctx, Text("hello"))

			So(payload, ShouldBeNil)
			So(err, ShouldEqual, context.Canceled)
		})
	})
}

func TestFireAndForget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame)

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), 16)

		// RQ -> RS: REQUEST_FNF
		Convey("When send a FireAndForget request", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)
			defer wg.Wait()

			go func() {
				defer wg.Done()

				Convey("The FireAndForget request should be sent", t, func() {
					f, ok := <-frameSender

					So(ok, ShouldBeTrue)
					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 1)
					So(f.Type(), ShouldEqual, frame.TypeRequestFireAndForget)
					So(f.Flags(), ShouldEqual, frame.FlagMetadata)

					requestFrame := f.(*frame.RequestFireAndForgetFrame)

					So(requestFrame, ShouldNotBeNil)
					So(string(requestFrame.Data), ShouldEqual, "hello")
					So(string(requestFrame.Metadata), ShouldEqual, "world")
				})
			}()

			So(requester.FireAndForget(ctx, Text("hello").WithMetadata([]byte("world"))), ShouldBeNil)
		})
	})
}

func TestMetadataPush(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	frameSender := make(chan frame.Frame)

	Convey("Given a client requester", t, func() {
		requester := NewRequester(logger, frameSender, ClientStreamIDs(), 16)

		Convey("When send a MetadataPush request", func() {
			wg := new(sync.WaitGroup)
			wg.Add(1)

			go func() {
				defer wg.Done()

				Convey("The MetadataPush frame should be sent", t, func() {
					f, ok := <-frameSender

					So(ok, ShouldBeTrue)
					So(f, ShouldNotBeNil)
					So(f.StreamID(), ShouldEqual, 0)
					So(f.Type(), ShouldEqual, frame.TypeMetadataPush)
					So(f.Flags(), ShouldEqual, frame.FlagMetadata)

					frame := f.(*frame.MetadataPushFrame)

					So(string(frame.Metadata), ShouldEqual, "hello")
				})
			}()

			So(requester.MetadataPush(ctx, []byte("hello")), ShouldBeNil)

			wg.Wait()
		})
	})
}
