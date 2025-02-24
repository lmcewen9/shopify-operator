package scraper

import (
	"encoding/json"
	"fmt"
	"log"
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
		log.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	var pResp ProductResponse

	if err := json.NewDecoder(res.Body).Decode(&pResp); err != nil {
		log.Fatal(err)
	}

	if len(pResp.Products) == 0 {
		return ret, nil
	}

	for i := 0; i < len(pResp.Products); i++ {
		ret = append(ret, pResp.Products[i].TextOutput(config))
	}
	return ret, nil

}
