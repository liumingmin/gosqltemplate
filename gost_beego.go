package gost

import (
	"time"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego"

	"github.com/astaxie/beego/cache"

	_ "github.com/go-sql-driver/mysql"
	"fmt"
)

type BeegoOrm struct{
	MemCacheMgr cache.Cache
	Ormer orm.Ormer
}

func init(){
	connStr := "root:123456@tcp(127.0.0.1:3306)/test_debug?charset=utf8&parseTime=true&loc=Asia%2FShanghai"
	orm.RegisterDriver("mysql", orm.DRMySQL)
	err := orm.RegisterDataBase("default", "mysql", connStr)

	if err != nil {
		beego.BeeLogger.Error("ConnStr=%v, err=", connStr, err)
	} else {
		beego.BeeLogger.Info("ConnStr=%v", connStr)
	}

	orm.SetMaxIdleConns("default",2)
	orm.SetMaxOpenConns("default",10)
	//orm.DefaultTimeLoc = time.UTC

	//err = orm.RunSyncdb("default", false, true)
}

func NewOrm(dbname string)orm.Ormer{

	var result = orm.NewOrm()
	result.Using(dbname)

	return result
}

func (borm *BeegoOrm) QueryCacheDelete(modelName string, cacheByValue string){
	CacheDeleteWrap(borm,modelName,cacheByValue)
}

func (borm *BeegoOrm)  QueryValuesByMap( queryId string, paramMap map[string]string, cacheTime time.Duration) (entities *QueryResult,err error){
	return QueryValuesWrap(borm,false,queryId,paramMap,cacheTime)
}

func (borm *BeegoOrm) QueryValueListByMap(queryId string, paramMap map[string]string, cacheTime time.Duration) (entities *QueryResult, err error){
	return QueryValuesWrap(borm,true,queryId,paramMap,cacheTime)
}

func (borm *BeegoOrm)  LogInfo(format string, a ...interface{}){
	fmt.Printf(format+"\n", a...)
}

func (borm *BeegoOrm)  LogDebug(format string, a ...interface{}){
	fmt.Printf(format+"\n", a...)
}

func (borm *BeegoOrm)  LogError(format string, a ...interface{}){
	fmt.Printf(format+"\n", a...)
}

func (borm *BeegoOrm)CacheClearAll() error{
	return borm.MemCacheMgr.ClearAll()
}

func (borm *BeegoOrm) cacheGet(key string) interface{}{
	return borm.MemCacheMgr.Get(key)
}

func (borm *BeegoOrm) cachePut(key string, val interface{}, timeout time.Duration) error{
	return borm.MemCacheMgr.Put(key,val,timeout)
}

func (borm *BeegoOrm) cacheDelete(key string) error{
	return borm.MemCacheMgr.Delete(key)
}

func (borm *BeegoOrm)cacheIsExist(key string) bool{
	return borm.MemCacheMgr.IsExist(key)
}


func (borm *BeegoOrm) rawQueryCount(sql string) (totalCount int64,err error){
	err = borm.Ormer.Raw(sql).QueryRow(&totalCount)
	return
}

func (borm *BeegoOrm) rawQueryValues(sql string) (retvalues interface{},err error){
	var values []orm.Params
	_,err = borm.Ormer.Raw(sql).Values(&values)
	retvalues = values
	return
}

func (borm *BeegoOrm) rawQueryValueList(sql string) (retvalues interface{},err error){
	var lists []orm.ParamsList
	_, err =  borm.Ormer.Raw(sql).ValuesList(&lists)
	retvalues = lists
	return
}

