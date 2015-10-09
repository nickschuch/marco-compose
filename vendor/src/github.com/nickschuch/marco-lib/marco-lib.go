package marco

import (
        "bytes"
        "encoding/json"
        "net/http"
)

type Backend struct {
        Type   string
        Domain string
        List   []string
}

func Send(backends []Backend, url string) error {
        b, err := json.Marshal(backends)
        if err != nil {
                return err
        }

        req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
        req.Header.Set("Content-Type", "application/json")

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
                return err
        }
        defer resp.Body.Close()

        return nil
}
