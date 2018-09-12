package gost

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	gSqlTemplateDir string

	gQueryInfos     = make(map[string]*QueryInfo)
	gQueryInfoMutex = new(sync.Mutex)

	gTableAndKeyCacheMap      = make(map[string](map[string]bool))
	gTableAndKeyCacheMapMutex = new(sync.Mutex)
)

func init() {
	SetSqlTemplateDir("")

	LoadStConfFiles()
}

type AnyOrm interface {
	QueryCacheDelete(modelName string, cacheByValue string)

	QueryValuesByMap(queryId string, paramMap map[string]string, cacheTime time.Duration) (entities *QueryResult, err error)

	QueryValueListByMap(queryId string, paramMap map[string]string, cacheTime time.Duration) (entities *QueryResult, err error)

	// clear all cache.
	CacheClearAll() error

	LogInfo(format string, a ...interface{})

	LogDebug(format string, a ...interface{})

	LogError(format string, a ...interface{})

	cacheGet(key string) interface{}
	// set cached value with key and expire time.
	cachePut(key string, val interface{}, timeout time.Duration) error
	// delete cached value by key.
	cacheDelete(key string) error
	// check if cached value exists or not.
	cacheIsExist(key string) bool

	//
	rawQueryCount(sql string) (int64, error)

	//ptr,[{"f1":""},{"f2":""}]
	rawQueryValues(sql string) (interface{}, error)

	//ptr,[[v1,v2],[v1,v2]]
	rawQueryValueList(sql string) (interface{}, error)
}

type QueryResult struct {
	TotalCount int64
	ResultList interface{}
}

///////////////////////////////////////////////////////////////////////////////////////////////
type QueryInfoConf struct {
	QueryInfos []QueryInfo `xml:"QueryInfo"`
}

type QueryInfo struct {
	Id            string       `xml:"Id"`
	RefModelNames string       `xml:"RefModelNames"`
	Cache         string       `xml:"Cache"`
	CacheBy       string       `xml:"CacheBy"`
	SQL           string       `xml:"SQL"`
	BindParams    []BindParams `xml:"BindParams"`
	Remark        string       `xml:"Remark"`
}

type BindParamGroup struct {
	BindParams []BindParam `xml:"BindParam"`
	ConnSymbol string      `xml:"ConnSymbol,attr"`
}

type BindParam struct {
	FieldExpress string `xml:"FieldExpress"`
	FormName     string `xml:"FormName"`
	ConnSymbol   string `xml:"ConnSymbol,attr"`
}

type BindParams struct {
	BindParamGroups []BindParamGroup `xml:"BindParamGroup"`
	RawSymbol       string           `xml:"RawSymbol,attr"`
	RawExpress      string           `xml:"RawExpress,attr"`
}

////////////////////////////////////////////////////////////////////////////////////////////////////
func SetSqlTemplateDir(sqlTmpDir string) {
	gSqlTemplateDir = sqlTmpDir
	if strings.TrimSpace(gSqlTemplateDir) == "" {
		file, _ := exec.LookPath(os.Args[0])
		fdir := filepath.Dir(file)
		gSqlTemplateDir = filepath.Join(fdir, "stconf")
	}
}

func PathExist(filename string) bool {
	_, err := os.Stat(filename)

	return err == nil || os.IsExist(err)
}

func Md5Hash(content string) string {
	hash := md5.New()
	hash.Write([]byte(content))
	md := hash.Sum(nil)
	return base64.StdEncoding.EncodeToString(md)
}

func deSqlInject(paramVal string) string {
	paramVal = strings.Replace(paramVal, "'", "", -1)
	paramVal = strings.Replace(paramVal, "--", "", -1)
	return paramVal
}

func getStConfFile() []string {
	files := make([]string, 0, 10)

	if !PathExist(gSqlTemplateDir) {
		return files
	}

	filepath.Walk(gSqlTemplateDir, func(filename string, fi os.FileInfo, err error) error { //遍历目录
		if fi.IsDir() { // 忽略目录
			return nil
		}

		if strings.HasSuffix(strings.ToLower(fi.Name()), ".gost") {
			files = append(files, filename)
		}

		return nil
	})

	return files
}

func logStd(format string, a ...interface{}) {
	fmt.Printf(format+"\n", a...)
}

func LoadStConfFiles() {
	logStd("Begin load query info,from :%v", gSqlTemplateDir)

	stFiles := getStConfFile()
	for _, stFile := range stFiles {
		content, err := ioutil.ReadFile(stFile)
		if err != nil {
			logStd("%v", err)
			continue
		}

		var queryInfoConf QueryInfoConf
		err = xml.Unmarshal(content, &queryInfoConf)
		if err != nil {
			logStd("%v", err)
			continue
		}

		if len(queryInfoConf.QueryInfos) == 0 {
			logStd("no query infos,file: %v", stFile)
			continue
		}

		for _, queryInfo := range queryInfoConf.QueryInfos {
			if _, isOk := gQueryInfos[queryInfo.Id]; isOk {
				logStd(" query info duplicate, will override,id: %v", queryInfo.Id)
			}
			gQueryInfos[queryInfo.Id] = &queryInfo
		}
	}

	logStd("End load query info,count :%v", len(gQueryInfos))
}

func genQuerySql(orm AnyOrm, queryInfo *QueryInfo, paramMap map[string]string) (string, bool) {
	if len(queryInfo.BindParams) == 0 {
		return queryInfo.SQL, true
	}

	searchapis := make([]interface{}, 0, 1)
	for _, bds := range queryInfo.BindParams {
		var searchapi string = ""

		if bds.RawSymbol == "" {
			for _, group := range bds.BindParamGroups {

				var groupclause string = ""

				for _, v := range group.BindParams {
					var paramVal string = ""
					if v, ok := paramMap[v.FormName]; ok {
						paramVal = v
					}

					paramVal = deSqlInject(paramVal)

					if paramVal != "" {
						if groupclause == "" {
							groupclause += " " + fmt.Sprintf(v.FieldExpress, paramVal)
						} else {
							groupclause += "  " + v.ConnSymbol + " " + fmt.Sprintf(v.FieldExpress, paramVal)
						}
					}
				}

				if groupclause != "" {
					searchapi += " " + group.ConnSymbol + " (" + groupclause + ")"
				}
			}
		} else {
			var paramVal string = ""
			if v, ok := paramMap[bds.RawSymbol]; ok {
				paramVal = v
			}

			paramVal = deSqlInject(paramVal)

			paramVal = fmt.Sprintf(bds.RawExpress, paramVal)

			searchapi = paramVal
		}

		searchapis = append(searchapis, searchapi)
	}

	sqlstr := fmt.Sprintf(queryInfo.SQL, searchapis...)

	orm.LogDebug("Gen sql %v", sqlstr)

	return sqlstr, true
}

//queryId用来查出关联了那些表名RefModelNames, cacheByValue 需要按字段时，对应字段的值，为空则是缓存到表
func cachePut(orm AnyOrm, modelNames []string, cacheByValue string, condKey string, val interface{}, timeout time.Duration) error {
	gTableAndKeyCacheMapMutex.Lock()

	for _, modelName := range modelNames {
		if modelName == "" {
			continue
		}

		conds, isok := gTableAndKeyCacheMap[modelName+cacheByValue]
		if !isok {
			conds = make(map[string]bool)
		}

		conds[condKey] = true
		gTableAndKeyCacheMap[modelName+cacheByValue] = conds
	}

	gTableAndKeyCacheMapMutex.Unlock()

	if timeout == 0 {
		timeout = 60 * time.Second
	}
	return orm.cachePut(condKey, val, 60)
}

func queryContentByCond(orm AnyOrm, retList bool, queryInfo *QueryInfo, paramMap map[string]string) (*QueryResult, error) {
	sqlstr, _ := genQuerySql(orm, queryInfo, paramMap)
	var err error = nil
	var totalSize int64 = 0

	var pStartStr string = ""
	var pLimitStr string = ""
	if v, ok := paramMap["__start"]; ok {
		pStartStr = v
	}

	if v, ok := paramMap["__limit"]; ok {
		pLimitStr = v
	}

	pStart, _ := strconv.Atoi(pStartStr)
	pLimit, _ := strconv.Atoi(pLimitStr)

	if pLimit > 0 {
		if pStart == 0 {
			cntSql := " select count(*) from (" + sqlstr + ") __t__"
			totalSize, err = orm.rawQueryCount(cntSql)

			fmt.Println(cntSql)
		}
		sqlstr += fmt.Sprintf(" limit %d,%d ", pStart, pLimit)
	}

	qr := &QueryResult{}
	qr.TotalCount = totalSize

	//fmt.Println(sqlstr)

	if retList {
		qr.ResultList, err = orm.rawQueryValueList(sqlstr)
	} else {
		qr.ResultList, err = orm.rawQueryValues(sqlstr)
	}

	return qr, err
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//按表和按字段清除缓存
func CacheDeleteWrap(orm AnyOrm, modelName string, cacheByValue string) {
	gTableAndKeyCacheMapMutex.Lock()
	defer gTableAndKeyCacheMapMutex.Unlock()

	if conds, isok := gTableAndKeyCacheMap[modelName+cacheByValue]; isok {
		for cond := range conds {
			orm.cacheDelete(cond)
		}

		//delete(gTableAndKeyCacheMap,modelName+cacheByValue)
	}
}

func QueryValuesWrap(orm AnyOrm, retList bool, queryId string, paramMap map[string]string, cacheTime time.Duration) (entities *QueryResult, err error) {
	var queryInfo *QueryInfo = nil

	gQueryInfoMutex.Lock()
	if v, isOk := gQueryInfos[queryId]; isOk {
		queryInfo = v
	}
	gQueryInfoMutex.Unlock()

	if queryInfo == nil {
		return nil, errors.New("not found query by id:" + queryId)
	}

	if queryInfo.Cache != "no" {
		condArray := make([]string, 0, 1)
		for condk, condv := range paramMap {
			condArray = append(condArray, condk)
			condArray = append(condArray, condv)
		}
		sort.Strings(condArray)
		condKey := queryId + "-" + strings.Join(condArray, "-")

		md5key := Md5Hash(condKey)
		if orm.cacheIsExist(md5key) {
			entities = orm.cacheGet(md5key).(*QueryResult)
		}

		if entities == nil {
			entities, err = queryContentByCond(orm, retList, queryInfo, paramMap)

			if err == nil { //modelNames []string
				modelNames := strings.Split(queryInfo.RefModelNames, ",")

				cacheByValue := ""
				if v, isOk := paramMap[queryInfo.CacheBy]; isOk {
					cacheByValue = v
				}

				cachePut(orm, modelNames, cacheByValue, md5key, entities, cacheTime)
			}
		}
	} else {
		entities, err = queryContentByCond(orm, retList, queryInfo, paramMap)
	}

	return
}
