package dl

import (
  "net/http"
  "os"
)

type Dl struct {
   taskNum uint16;
}

func Init() {
  Reset()
}

func Reset() {

}

func (*Dl) download() {

}

func (this *Dl) AddTask(req http.Request, file os.File) {
  client := http.Client{}

  go this.download();

  client.Do(&req)
}
