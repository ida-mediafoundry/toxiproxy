package toxics

import (
    "math/rand"
	"bufio"
	"bytes"
	"io"
    "io/ioutil"
	"net/http"
    "encoding/json"

	"github.com/Shopify/toxiproxy/stream"
)

type HttpToxic struct{
    Location string `json:"location"`
}

type HttpToxicResponseBody struct {
    Status  int     `json:"status"`
    Message string  `json:"message"`
}

func (t *HttpToxic) ModifyResponse(resp *http.Response) {
    location := t.Location
    body := &HttpToxicResponseBody {}
    doError := len(location) == 0 || rand.Intn(2) != 0

    if doError {
        body.Status = 500
        body.Message = "500 Internal Server Error!"
    } else {
        body.Status = 302
        body.Message = "302 Temporary redirect"
        resp.Header.Set("Location", location)
    }

    bodyRes, _ := json.Marshal(body)
    bodyStr := string(bodyRes)

    resp.StatusCode = body.Status
    resp.Status = body.Message
    resp.ContentLength = int64(len(bodyStr))
    resp.Body = ioutil.NopCloser(bytes.NewBufferString(bodyStr))
}

func (t *HttpToxic) Pipe(stub *ToxicStub) {
	buffer := bytes.NewBuffer(make([]byte, 0, 32*1024))
	writer := stream.NewChanWriter(stub.Output)
	reader := stream.NewChanReader(stub.Input)
	reader.SetInterrupt(stub.Interrupt)
	for {
		tee := io.TeeReader(reader, buffer)
		resp, err := http.ReadResponse(bufio.NewReader(tee), nil)
		if err == stream.ErrInterrupted {
			buffer.WriteTo(writer)
			return
		} else if err == io.EOF {
			stub.Close()
			return
		}
		if err != nil {
			buffer.WriteTo(writer)
		} else {
			t.ModifyResponse(resp)
			resp.Write(writer)
		}
		buffer.Reset()
	}
}

func init() {
	Register("http", new(HttpToxic))
}
