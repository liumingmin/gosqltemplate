package gost

import (
	"testing"
	"github.com/astaxie/beego/cache"
	"time"
	"fmt"
)

func TestBeegoOrm_RawQueryValues(t *testing.T) {
	borm := &BeegoOrm{}
	borm.MemCacheMgr, _ = cache.NewCache("memory", `{"interval":60}`)
	borm.Ormer = NewOrm("default")

	param3 := make(map[string]string)
	param3["UserId"]="ddddd"
	param3["__start"]="10"
	param3["__limit"]="20"
	v,err := borm.QueryByCond("query_1101",param3,60*time.Second)

	fmt.Println(v,err)
}

