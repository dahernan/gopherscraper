package scraper

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/url"

	"github.com/dahernan/gopherscraper/elasticsearch"
	"github.com/dahernan/gopherscraper/model"
)

type StorageItems interface {
	StoreItem(it ItemResult)
}

type ScrapAndStoreItems interface {
	ScrapAndStore(selector ScrapSelector) (string, error)
}

// Elastic Search storage
type ElasticStorage struct {
	redis       *RedisScrapdata
	elasticItem *elastic.ItemElastic
}

func NewElasticStorage() StorageItems {
	return ElasticStorage{
		redis:       NewRedisScrapdata(),
		elasticItem: elastic.NewItemElastic("gopherscrap"),
	}
}

func (sto ElasticStorage) StoreItem(it ItemResult) {
	if it.Err != nil {
		log.Printf("ERROR Scrap [%v]:ElasticStorage with Item, with message %v\n", it.JobId, it.Err.Error())
		return
	}

	var err error
	var resp elastic.ElasticResponse

	index := itemIndex(it)
	it.Item.Index = index

	resp, err = sto.elasticItem.Put(&it.Item)

	if err != nil {
		log.Printf("ERROR Scrap [%v]:ElasticStorage in PUT the item %v into ES, with message %v", it.JobId, it.Item.Id, err.Error())
	} else {
		log.Printf("INFO: Scrap [%v]:ElasticStorage PUT index %v type: %v Id: %v", it.JobId, resp.Index, resp.Type, resp.Id)
	}

}

// Redis Storage
type RedisStorage struct {
	redis *RedisScrapdata
}

func NewRedisStorage() StorageItems {
	return RedisStorage{
		redis: NewRedisScrapdata(),
	}
}

func (sto RedisStorage) StoreItem(it ItemResult) {
	jobKey := scrapJobsKey(it.JobId)
	jobKeyMeta := scrapJobsKeyMeta(it.JobId)

	if it.Err != nil {
		log.Printf("ERROR Scrap [%v] RedisStorage:StoreItems with Item, with message %v", it.JobId, it.Err.Error())
		sto.redis.client.HIncrBy(jobKeyMeta, "errors", 1)
		sto.redis.client.HSet(jobKeyMeta, "lastError", it.Err.Error())
		return
	}
	index := itemIndex(it)
	it.Item.Index = index

	b, err := json.Marshal(&it.Item)
	if err != nil {
		b = []byte{}
		log.Printf("ERROR:[Scraper:RedisStorage] can not marshall item %v %v", it.Item.Id, err.Error())
	}

	sto.redis.client.HSet(jobKey, index, string(b))
	sto.redis.client.HIncrBy(jobKeyMeta, "items", 1)

	defer sto.redis.client.Expire(jobKey, 60*10)     // 10 minutes
	defer sto.redis.client.Expire(jobKeyMeta, 60*10) // 10 minutes

}

// Local Files Storage
type FileStorage struct {
	baseDir string
}

func NewFileStorage() StorageItems {
	return FileStorage{
		baseDir: "/tmp/items",
	}
}

func (sto FileStorage) StoreItem(it ItemResult) {
	if it.Err != nil {
		return
	}

	index := itemIndex(it)
	it.Item.Index = index

	WriteJsonToDisk(sto.baseDir, it.Item)
}

// scrap and store
type DefaultScrapAndStore struct {
	scrapper ScrapperItems
	storages []StorageItems
}

func NewScrapAndStore(sc ScrapperItems, storages []StorageItems) ScrapAndStoreItems {
	return DefaultScrapAndStore{
		scrapper: sc,
		storages: storages,
	}
}

func NewElasticScrapAndStore() ScrapAndStoreItems {
	return DefaultScrapAndStore{
		scrapper: NewRecursiveScrapper(),
		storages: []StorageItems{NewElasticStorage(), NewRedisStorage(), NewFileStorage()},
	}
}

func (ss DefaultScrapAndStore) ScrapAndStore(selector ScrapSelector) (string, error) {
	rdata := NewRedisScrapdata()
	rdata.SaveSelector(selector)

	jobId, items, err := ss.scrapper.Scrap(selector)
	if err != nil {
		return jobId, err
	}

	go ss.Store(items)

	return jobId, nil
}

func (ss DefaultScrapAndStore) Store(items chan ItemResult) {
	for it := range items {
		for i, _ := range ss.storages {
			go ss.storages[i].StoreItem(it)
		}
	}
}

func WriteJsonToDisk(baseDir string, it model.Item) {
	if it.Id == "" {
		log.Printf("ERROR: [Scraper:WriteJsonToDisk] write to disk a item with empty id %v", it.Link)
		return
	}

	filename := baseDir + "/" + it.Id + ".json"

	b, err := json.Marshal(&it)
	if err != nil {
		log.Printf("ERROR:[Scraper:WriteJsonToDisk] can not marshall item %v %v", it.Id, err.Error())
		return
	}
	err = ioutil.WriteFile(filename, b, 0644)
	if err != nil {
		log.Printf("ERROR: [Scraper:WriteJsonToDisk] can not write file %v %v", filename, err.Error())
		return
	}
}

func itemIndex(it ItemResult) string {
	if it.Err != nil {
		return ""
	}
	item := it.Item

	u, err := url.Parse(item.ScrapUrl)
	if err != nil {
		return ""
	}
	return u.Host + "/" + item.Id
}
