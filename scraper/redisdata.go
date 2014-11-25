package scraper

import (
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dahernan/gopherscraper/model"
	"github.com/dahernan/gopherscraper/redis"
)

const (
	scrapSelectorKeyPrefix = "scrapSelector"
	scrapJobsKeyPrefix     = "scrapJobs"
	scrapLogKeyPrefix      = "scrapLog"
)

var (
	ErrSelectorNotFound = errors.New("Selector not found")
	ErrJobNotFound      = errors.New("Scrap job not found")
)

type RedisScrapdata struct {
	client *redis.RedisClient
}

func NewRedisScrapdata() *RedisScrapdata {
	return &RedisScrapdata{redis.Redis()}
}

func (r *RedisScrapdata) SaveSelector(s ScrapSelector) error {
	err := validateSelector(s)
	if err != nil {
		return err
	}

	o, err := json.Marshal(s)
	if err != nil {
		return err
	}

	u, err := url.Parse(s.Url)
	if err != nil {
		return err
	}

	stype := strings.Replace(s.Stype, " ", "", -1)
	if stype == "" {
		stype = SelectorTypeList
	}
	if (stype != SelectorTypeList) && (stype != SelectorTypeDetail) {
		stype = SelectorTypeList
	}

	_, err = r.client.HSet(scrapSelectorKeyPrefix, u.Host+":"+stype, string(o))
	return err
}

func (r *RedisScrapdata) Selector(scrapUrl, stype string) (ScrapSelector, error) {
	var s ScrapSelector

	u, err := url.Parse(scrapUrl)
	if err != nil {
		return s, err
	}

	if stype == "" {
		stype = SelectorTypeList
	}

	data, err := r.client.HGet(scrapSelectorKeyPrefix, u.Host+":"+stype)
	if err != nil {
		return s, err
	}
	if len(data) <= 0 {
		return s, ErrSelectorNotFound
	}

	err = json.Unmarshal(data, &s)
	// set the original url
	s.Url = scrapUrl

	return s, err
}

func (r *RedisScrapdata) StartJob(jobId string, s ScrapSelector) error {
	jobKey := scrapJobsKey(jobId)
	jobKeyMeta := scrapJobsKeyMeta(jobId)

	defer r.client.Expire(jobKey, 60*10)
	defer r.client.Expire(jobKeyMeta, 60*60*24)

	unixTime := time.Now().Unix()

	r.client.HIncrBy(jobKeyMeta, "totalHits", 1)
	r.client.HSet(jobKeyMeta, "start", strconv.FormatInt(unixTime, 10))
	r.client.HSet(jobKeyMeta, "url", s.Url)

	r.client.HDel(jobKeyMeta, "hits:")
	r.client.HDel(jobKeyMeta, "items")
	r.client.HDel(jobKeyMeta, "finish")
	r.client.HDel(jobKeyMeta, "errors")
	r.client.HDel(jobKeyMeta, "lastError")

	return nil
}

func (r *RedisScrapdata) FinishJob(jobId string) error {
	jobKey := scrapJobsKey(jobId)
	jobKeyMeta := scrapJobsKeyMeta(jobId)

	defer r.client.Expire(jobKey, 60*10)
	defer r.client.Expire(jobKeyMeta, 60*60*24)

	unixTime := strconv.FormatInt(time.Now().Unix(), 10)
	r.client.HSet(jobKeyMeta, "finish", unixTime)

	return nil
}

func (r *RedisScrapdata) ScrapJob(jobId string) (map[string]interface{}, error) {
	result := map[string]interface{}{}

	jobKey := scrapJobsKey(jobId)
	jobKeyMeta := scrapJobsKeyMeta(jobId)

	meta, err := r.client.HGetAll(jobKeyMeta)
	if err != nil {
		return nil, err
	}

	var items []model.Item

	itemsMap, err := r.client.HGetAll(jobKey)
	if err != nil {
		return nil, err
	}

	for k, _ := range itemsMap {
		var it model.Item
		json.Unmarshal([]byte(itemsMap[k]), &it)
		items = append(items, it)
	}

	if len(meta) == 0 && len(items) == 0 {
		return result, ErrJobNotFound
	}

	result["meta"] = meta
	result["items"] = items

	return result, nil
}

func (r *RedisScrapdata) ScrapLog() []string {
	logKey := scrapLogKey()
	r.ScrapLogTrim()
	log, _ := r.client.LRange(logKey, 0, -1)
	return log
}

func (r *RedisScrapdata) ScrapLogWrite(line string) {
	logKey := scrapLogKey()
	r.client.LPush(logKey, line)
}

func (r *RedisScrapdata) ScrapLogTrim() {
	logKey := scrapLogKey()
	r.client.LTrim(logKey, 0, 40)
}

func scrapLogKey() string {
	return scrapLogKeyPrefix
}

func scrapJobsKey(jobId string) string {
	return scrapJobsKeyPrefix + ":" + jobId
}

func scrapJobsKeyMeta(jobId string) string {
	return scrapJobsKey(jobId) + ":meta"
}
