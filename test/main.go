package main

import (
	"time"
	"github.com/astaxie/beego/cache"
	"fmt"
	"github.com/liumingmin/gosqltemplate"
)

func main(){
	borm := &gost.BeegoOrm{}
	borm.MemCacheMgr, _ = cache.NewCache("memory", `{"interval":60}`)
	borm.Ormer = gost.NewOrm("default")

	param3 := make(map[string]string)
	param3["UserId"]="ddddd"
	param3["__start"]="10"
	param3["__limit"]="20"
	v,err := borm.QueryValuesByMap("query_1101",param3,60*time.Second)

	fmt.Println(v,err)


	xxorm :=  &gost.XormOrm{}
	xxorm.MemCacheMgr, _ = cache.NewCache("memory", `{"interval":60}`)
	xxorm.Ormer = gost.NewXOrm()

	v,err = xxorm.QueryValuesByMap("query_1101",param3,60*time.Second)

	fmt.Println(v,err)
}
