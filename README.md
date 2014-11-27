
# Rest Scrapping 

## Scrap an amazon product with a selector

```
$ curl -XPOST http://localhost:3001/api/scraper/scrap -d '{
  "url": "http://www.amazon.co.uk/gp/product/B00HZH5ESO",  
  "base": ".a-container",
  "stype": "detail",
  "IdFrom": "IdFromLink",  
  "IdExtractor": {
    "urlPathIndex": -1
  },  
  "image": {
    "exp": "#imgTagWrapperId img",
    "attr": "data-old-hires"
  },
  "title": {
    "exp": "#productTitle"
  },
  "description": {
    "exp": "#feature-bullets"
  },
  "price": {
    "exp": "#priceblock_ourprice"
  }  
}'
```

## Returns the jobId
```
{
  "jobId": "D2277869965"
}
```

## Gets the Job details
```
$ curl -XGET http://localhost:3001/api/scraper/job/D2277869965
{
  "items": [
    {
      "id": "B00HZH5ESO",
      "link": "http://www.amazon.co.uk/gp/product/B00HZH5ESO",
      "image": "http://ecx.images-amazon.com/images/I/61JJ1tQBjOL._SL1051_.jpg",
      "title": "Fujifilm FinePix S1 Digital Camera (16.4MP, 50x Optical Zoom) 3 Inch LCD",
      "description": "Robust dust and weather-resistant bridge camera50x image stabilised optical zoom lens (24-1200mm equivalent)1/2.3-inch 16.4 megapixel backlit CMOS sensorWi-fi connectivity",
      "price": 301.24,
      "currency": "£",
      "scrapUrl": "http://www.amazon.co.uk/gp/product/B00HZH5ESO",
      "index": "www.amazon.co.uk/B00HZH5ESO",
      "lastScrap": "2014-11-27T17:30:52Z"
    }
  ],
  "meta": {
    "finish": "1417109452",
    "items": "1",
    "start": "1417109451",
    "totalHits": "2",
    "url": "http://www.amazon.co.uk/gp/product/B00HZH5ESO"
  }
}
```

## Scrap another amazon product does not need the selector because is saved in Redis
```
$ curl -XPOST http://localhost:3001/api/scraper/scrap -d '{
  "url": "http://www.amazon.co.uk/gp/product/B00AQBWNXA"
}'

{
  "jobId": "D3686865129"
}
```
```
$ curl -XGET http://localhost:3001/api/scraper/job/D2277869965

 {
  "items": [
    {
      "id": "B00AQBWNXA",
      "link": "http://www.amazon.co.uk/gp/product/B00AQBWNXA",
      "image": "http://ecx.images-amazon.com/images/I/71bgtNPp+OL._SL1500_.jpg",
      "title": "BenQ GW2760HM LED VA Panel 27-inch W Multimedia Monitor 1920 x 1080 20M:1, 4 ms GTG, DVI, HDMI \u0026 Speakers - Glossy Black",
      "description": "\n\u0009\n\u0009\u0009\n\u0009\u0009\u0009\n\u0009\u0009\u0009\n\u0009\u0009\u0009\u0009\n\u0009\u0009\u0009\u0009\u0009 Flicker-free backlight for visual pleasureReading mode for an optimised reading experienceHDMI cable includedFull HD 1080 p 16:9 visual display20M:1 dynamic contrast ratio for depth and definition \n\u0009\u0009\u0009\u0009\n\u0009\u0009\u0009\n\u0009\u0009\n\u0009\u0009\n\u0009\u0009\n\u0009\u0009\n\u0009\u0009\n\u0009\u0009\n\u0009\n",
      "price": 154.98,
      "currency": "£",
      "scrapUrl": "http://www.amazon.co.uk/gp/product/B00AQBWNXA",
      "index": "www.amazon.co.uk/B00AQBWNXA",
      "lastScrap": "2014-11-27T17:39:03Z"
    }
  ],
  "meta": {
    "finish": "1417109943",
    "items": "1",
    "start": "1417109942",
    "totalHits": "1",
    "url": "http://www.amazon.co.uk/gp/product/B00AQBWNXA"
  }
}
```


# Search in ElasticSearch index

```
curl -XGET "http://localhost:3001/api/search/web/www.amazon.co.uk" -d'
{
         "query": {
            "match_all": {}
         }
}' 

{
  "_shards": {
    "failed": 0,
    "successful": 5,
    "total": 5
  },
  "hits": {
    "hits": [
      {
        "_id": "B00AQBWNXA",
        "_index": "gopherscrap",
        "_score": 1,
        "_source": {
        ...

  ...

  ...

```


