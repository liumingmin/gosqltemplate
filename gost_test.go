package gost

import (
	"testing"
	//"fmt"
	"time"
)

func TestBaseOrm_QueryCacheDelete(t *testing.T) {
	ormer:= &BaseOrm{}
	param := make(map[string]string)
	param["Id"]="ddddd"
	ormer.QueryCacheDelete("Project","projectId")
}

func TestBaseOrm_RawQueryValues(t *testing.T) {
	ormer:= &BaseOrm{}
	param := make(map[string]string)
	param["UserId"]="ddddd"
	ormer.QueryByCond("query_1101",param,60*time.Second)


	param2 := make(map[string]string)
	param2["UserId"]="ddddd"
	param2["__start"]="0"
	param2["__limit"]="20"
	ormer.QueryByCond("query_1101",param2,60*time.Second)

	param3 := make(map[string]string)
	param3["UserId"]="ddddd"
	param3["__start"]="10"
	param3["__limit"]="20"
	ormer.QueryByCond("query_1101",param3,60*time.Second)
}