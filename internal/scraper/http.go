package scraper

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type Configuration struct {
	URL string
}

func FetchShopify(config *Configuration, page int) ([]string, error) {
	var ret []string
	var url string = fmt.Sprintf("https://%s/products.json?page=%s", config.URL, strconv.Itoa(page))
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := res.Body.Close(); err != nil {
			err = e
		}
	}()

	var pResp ProductResponse

	if err := json.NewDecoder(res.Body).Decode(&pResp); err != nil {
		return nil, err
	}

	if len(pResp.Products) == 0 {
		return ret, nil
	}

	for i := 0; i < len(pResp.Products); i++ {
		ret = append(ret, pResp.Products[i].TextOutput(config))
	}
	return ret, nil

}
