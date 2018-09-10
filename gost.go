package gost

import (
	"fmt"
	"strings"
	"os"
	"path/filepath"
	"os/exec"
	"io/ioutil"
	"encoding/xml"
	"sort"
	"time"
	"crypto/md5"
	"encoding/base64"
	"strconv"
	"sync"
	"github.com/pkg/errors"
)

var (
	sqlTemplateDir string
	globeQueryInfos = make(map[string]*QueryInfo)
    gQueryInfoMutex = new(sync.Mutex)
    gCacheModelInfoMutex = new(sync.Mutex)
    gModelNameCondCacheMap = make(map[string](map[string]bool))
)

func init(){
	SetSqlTemplateDir("")

	LoadStConfFiles()
}

type AnyOrm interface {
	CacheGet(key string) interface{}
	// set cached value with key and expire time.
	CachePut(key string, val interface{}, timeout time.Duration) error
	// delete cached value by key.
	CacheDelete(key string) error
	// check if cached value exists or not.
	CacheIsExist(key string) bool
	// clear all cache.
	CacheClearAll() error

	//ptr, [v1,v2,v3...]
	RawQueryCount(sql string) (int64, error)

	//ptr,[{"f1":""},{"f2":""}]
	RawQueryValues(sql string) (interface{}, error)

	//ptr,[[v1,v2],[v1,v2]]
	RawQueryValueList(sql string) (interface{}, error)
}


type BaseOrm struct{

}

type QueryResult struct{
	TotalCount int64
	ResultList interface{}
}
///////////////////////////////////////////////////////////////////////////////////////////////
type QueryInfoConf struct {
	QueryInfos []QueryInfo `xml:"QueryInfo"`
}

type QueryInfo struct {
	Id string `xml:"Id"`
	RefModelNames string `xml:"RefModelNames"`
	Cache string `xml:"Cache"`
	CacheBy string `xml:"CacheBy"`
	SQL string `xml:"SQL"`
	BindParams []BindParams `xml:"BindParams"`
	Remark string `xml:"Remark"`
}

type BindParamGroup struct {
	BindParams []BindParam `xml:"BindParam"`
	ConnSymbol string `xml:"ConnSymbol,attr"`
}

type BindParam struct {
	FieldExpress string `xml:"FieldExpress"`
	FormName string `xml:"FormName"`
	ConnSymbol string `xml:"ConnSymbol,attr"`
}

type BindParams struct {
	BindParamGroups []BindParamGroup `xml:"BindParamGroup"`
	RawSymbol string `xml:"RawSymbol,attr"`
	RawExpress string `xml:"RawExpress,attr"`
}

////////////////////////////////////////////////////////////////////////////////

func logWrap(format string, a ... interface{}){
	fmt.Printf(format+"\n",a...)
}

func SetSqlTemplateDir(sqlTmpDir string){
	sqlTemplateDir = sqlTmpDir
	if strings.TrimSpace(sqlTemplateDir) == ""{
		file, _ := exec.LookPath(os.Args[0])
		fdir := filepath.Dir(file)
		sqlTemplateDir = filepath.Join(fdir, "stconf")
	}
}

func PathExist(filename string) bool {
	_, err := os.Stat(filename)

	return err == nil || os.IsExist(err)
}

func Md5Hash(content string) string{
	hash := md5.New()
	hash.Write([]byte(content))
	md := hash.Sum(nil)
	return base64.StdEncoding.EncodeToString(md)
}

func deSqlInject(paramVal string) string {
	paramVal = strings.Replace(paramVal,"'","",-1)
	paramVal = strings.Replace(paramVal,"--","",-1)
	return paramVal
}

func getStConfFile() []string{
	files := make([]string, 0, 10)

	if !PathExist(sqlTemplateDir){
		return files
	}

	filepath.Walk(sqlTemplateDir, func(filename string, fi os.FileInfo, err error) error { //遍历目录
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

func LoadStConfFiles(){
	logWrap("Begin load query info,from :%v",sqlTemplateDir)

	stFiles := getStConfFile()
	for _,stFile := range stFiles {
		content, err := ioutil.ReadFile(stFile)
		if err!=nil{
			logWrap("%v",err)
			continue
		}

		var queryInfoConf QueryInfoConf
		err = xml.Unmarshal(content, &queryInfoConf)
		if err != nil {
			logWrap("%v",err)
			continue
		}

		if len(queryInfoConf.QueryInfos) == 0 {
			logWrap("no query infos,file: %v",stFile)
			continue
		}

		for _,queryInfo := range queryInfoConf.QueryInfos{
			if _,isOk := globeQueryInfos[queryInfo.Id];isOk{
				logWrap(" query info duplicate, will override,id: %v",queryInfo.Id)
			}
			globeQueryInfos[queryInfo.Id] = &queryInfo
		}
	}

	logWrap("End load query info,count :%v",len(globeQueryInfos))
}

func genQuerySql(queryInfo *QueryInfo, paramMap map[string]string) (string,bool){
	if len(queryInfo.BindParams) == 0{
		return queryInfo.SQL,true
	}

	searchapis := make([]interface{},0,1)
	for _,bds := range queryInfo.BindParams{
		var searchapi string = ""

		if bds.RawSymbol == "" {
			for _,group := range bds.BindParamGroups{

				var groupclause string = ""

				for _,v := range group.BindParams{
					var paramVal string = ""
					if v, ok := paramMap[v.FormName]; ok {
						paramVal = v
					}

					paramVal = deSqlInject(paramVal)

					if paramVal != "" {
						if groupclause == "" {
							groupclause += " " + fmt.Sprintf(v.FieldExpress, paramVal)
						}else{
							groupclause += "  "+v.ConnSymbol+" " + fmt.Sprintf(v.FieldExpress, paramVal)
						}
					}
				}

				if groupclause != ""{
					searchapi += " "+group.ConnSymbol +  " ("+ groupclause +")"
				}
			}
		}else {
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

	return sqlstr,true
}
//queryId用来查出关联了那些表名RefModelNames, cacheByValue 需要按字段时，对应字段的值，为空则是缓存到表
func cachePut(orm AnyOrm,modelNames []string, cacheByValue string, condKey string,val interface{}, timeout time.Duration)error{
	gCacheModelInfoMutex.Lock()

	for _,modelName := range modelNames{
		if modelName == ""{
			continue
		}

		conds,isok := gModelNameCondCacheMap[modelName+cacheByValue]
		if !isok{
			conds = make(map[string]bool)
		}

		conds[condKey] = true
		gModelNameCondCacheMap[modelName+cacheByValue] = conds
	}

	gCacheModelInfoMutex.Unlock()

	if timeout == 0{
		timeout = 60*time.Second
	}
	return orm.CachePut(condKey, val,60)
}

func queryContentByCond(orm AnyOrm, queryInfo *QueryInfo, paramMap map[string]string) (*QueryResult,error) {
	sqlstr,_ := genQuerySql(queryInfo, paramMap)
	var err error =nil
	var totalSize int64 = 0

	var pStartStr string = ""
	var pLimitStr string = ""
	if v, ok := paramMap["__start"]; ok {
		pStartStr = v
	}

	if v, ok := paramMap["__limit"]; ok {
		pLimitStr = v
	}

	pStart,_ := strconv.Atoi(pStartStr)
	pLimit,_ := strconv.Atoi(pLimitStr)

	if pLimit>0 {
		if pStart == 0{
			cntSql := " select count(*) from ("+sqlstr+") __t__"
			totalSize,err = orm.RawQueryCount(cntSql)
		}
		sqlstr += fmt.Sprintf(" limit %d,%d ",pStart,pLimit)
	}

	qr := &QueryResult{}
	qr.TotalCount = totalSize

	qr.ResultList,err = orm.RawQueryValues(sqlstr)

	return qr,err
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//按表和按字段清除缓存
func (orm *BaseOrm)QueryCacheDelete(modelName string, cacheByValue string){
	gCacheModelInfoMutex.Lock()
	defer gCacheModelInfoMutex.Unlock()

	if conds,isok := gModelNameCondCacheMap[modelName+cacheByValue];isok{
		for cond := range conds{
			orm.CacheDelete(cond)
		}

		//delete(gModelNameCondCacheMap,modelName+cacheByValue)
	}
}

func (orm *BaseOrm) QueryByCond( queryId string, paramMap map[string]string, cacheTime time.Duration) (entities interface{},err error) {
	var queryInfo *QueryInfo = nil

	gQueryInfoMutex.Lock()
	if v,isOk := globeQueryInfos[queryId];isOk{
		queryInfo = v
	}
	gQueryInfoMutex.Unlock()

	if queryInfo == nil{
		return nil,errors.New("not found query by id:"+queryId)
	}

	if queryInfo.Cache != "no"{
		condArray := make([]string,0,1)
		for condk,condv := range paramMap{
			condArray = append(condArray, condk)
			condArray = append(condArray, condv)
		}
		sort.Strings(condArray)
		condKey := queryId +"-"+ strings.Join(condArray,"-")

		md5key := Md5Hash(condKey)
		if orm.CacheIsExist(md5key) {
			entities = orm.CacheGet(md5key)
		}

		if entities == nil{
			entities,err = queryContentByCond(orm,queryInfo, paramMap)

			if err == nil {//modelNames []string
				modelNames := strings.Split(queryInfo.RefModelNames, ",")

				cacheByValue := ""
				if v,isOk :=paramMap[queryInfo.CacheBy];isOk{
					cacheByValue=v
				}

				cachePut(orm ,modelNames,cacheByValue, md5key, entities,cacheTime)
			}
		}
	}else{
		entities,err = queryContentByCond(orm,queryInfo, paramMap)
	}

	return
}

//待实现接口
func (orm *BaseOrm) CacheGet(key string) interface{}{
	return nil
}

func (orm *BaseOrm) CachePut(key string, val interface{}, timeout time.Duration) error{
	return nil
}

func (orm *BaseOrm) CacheDelete(key string) error{
	return nil
}

func (orm *BaseOrm)CacheIsExist(key string) bool{
	return false
}

func (orm *BaseOrm)CacheClearAll() error{
	return nil
}

func (orm *BaseOrm) RawQueryCount(sql string) (int64, error){
	return 0,nil
}

func (orm *BaseOrm) RawQueryValues(sql string) (interface{}, error){
	return nil,nil
}

func (orm *BaseOrm) RawQueryValueList(sql string) (interface{}, error){
	return nil,nil
}