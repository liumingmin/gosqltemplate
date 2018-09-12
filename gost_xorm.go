package gost

import (
	"github.com/astaxie/beego"
	"time"
	"github.com/go-xorm/xorm"
	"github.com/astaxie/beego/cache"
	"fmt"
	"github.com/pkg/errors"
)


type XormOrm struct{
	MemCacheMgr cache.Cache
	Ormer *xorm.Engine
}

func init(){


}

func NewXOrm()*xorm.Engine{

	engine, err := xorm.NewEngine("mysql", "root:123456@tcp(127.0.0.1:3306)/test_debug?charset=utf8")
	if err != nil {
		beego.BeeLogger.Error("ConnStr=%v, err=", "", err)
	} else {
		beego.BeeLogger.Info("ConnStr=%v", "")
	}

	return engine
}

func (xxorm *XormOrm) QueryCacheDelete(modelName string, cacheByValue string){
	CacheDeleteWrap(xxorm,modelName,cacheByValue)
}

func (xxorm *XormOrm)  QueryValuesByMap( queryId string, paramMap map[string]string, cacheTime time.Duration) (entities *QueryResult,err error){
	return QueryValuesWrap(xxorm,false,queryId,paramMap,cacheTime)
}

func (xxorm *XormOrm) QueryValueListByMap(queryId string, paramMap map[string]string, cacheTime time.Duration) (entities *QueryResult, err error){
	return QueryValuesWrap(xxorm,true,queryId,paramMap,cacheTime)
}

func (xxorm *XormOrm)  LogInfo(format string, a ...interface{}){
	fmt.Printf(format+"\n", a...)
}

func (xxorm *XormOrm)  LogDebug(format string, a ...interface{}){
	fmt.Printf(format+"\n", a...)
}

func (xxorm *XormOrm)  LogError(format string, a ...interface{}){
	fmt.Printf(format+"\n", a...)
}

func (xxorm *XormOrm)CacheClearAll() error{
	return xxorm.MemCacheMgr.ClearAll()
}

func (xxorm *XormOrm) cacheGet(key string) interface{}{
	return xxorm.MemCacheMgr.Get(key)
}

func (xxorm *XormOrm) cachePut(key string, val interface{}, timeout time.Duration) error{
	return xxorm.MemCacheMgr.Put(key,val,timeout)
}

func (xxorm *XormOrm) cacheDelete(key string) error{
	return xxorm.MemCacheMgr.Delete(key)
}

func (xxorm *XormOrm)cacheIsExist(key string) bool{
	return xxorm.MemCacheMgr.IsExist(key)
}


func (xxorm *XormOrm) rawQueryCount(sql string) (totalCount int64,err error){
	//err = xxorm.Ormer.QueryInterface(sql).QueryRow(&totalCount)
	return xxorm.Ormer.Count(sql)
}

func (xxorm *XormOrm) rawQueryValues(sql string) (retvalues interface{},err error){
	//var values []orm.Params
	//_,err = xxorm.Ormer.Raw(sql).Values(&values)
	//retvalues = values
	//return

	return  xxorm.Ormer.QueryInterface(sql)
}

func (xxorm *XormOrm) rawQueryValueList(sql string) (retvalues interface{},err error){
	//var lists []orm.ParamsList
	//_, err =  xxorm.Ormer.Raw(sql).ValuesList(&lists)
	//retvalues = lists
	//return
	retvalues = nil
	err = errors.New("xorm not support value lists ")
	xxorm.LogError("%v",err)
	return
}



