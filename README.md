# gosqltemplate
A golang sql template ,independent of database, small&amp; light.
简单的非侵入的sql模板库
##功能特性
*1、灵活的根据xml配置SQL模板动态生成SQL执行。
*2、支持分页，支持到表级别、字段级别缓存。
*3、非侵入式，本库不提供缓存库、ORM库，可方便嵌入已有项目或开发精简的微服务。

#执行查询

```Go
borm := &gost.BeegoOrm{}
borm.MemCacheMgr, _ = cache.NewCache("memory", `{"interval":60}`)
borm.Ormer = gost.NewOrm("default")

param3 := make(map[string]string)
param3["UserId"]="ddddd"
param3["__start"]="10"
param3["__limit"]="20"
v,err := borm.QueryValuesByMap("query_1101",param3,60*time.Second)

fmt.Println(v,err)
```


##1.编写模板SQL
```SQL
  select p.Id,p.Title,date_format(p.CreateDate,'%%Y-%%m-%%d') as Cdate,p.HadView,p.NeedView-p.HadView NotView,
	format(case when p.NeedView &gt;0 then p.HadView/(p.NeedView+0.0)*100 else 0 end,2) ViewRate,p.kind,p.isview,p.linkid  from
	(select t.Id,t.Title,t.CreateDate,t.HadView,t.NeedView,0 kind,1 isview,'' linkid from OaNotice t
    where 1=1 %s
    union all
    select t.Id,t.Title,t.CreateDate,t.HadView,t.NeedView,1 kind,l.IsView,l.Id linkid from OaNotice t,OaNoticeLink l
    where t.Id=l.NoticeId  %s ) p
    order by p.CreateDate desc
```
##2.配置查询条件，BindParam对应一个SQL条件，AND 或者OR，通过FormName从MAP中取到对应字段值拼到字段表达式FieldExpress中，形成一个SQL条件，然后通过BindParamGroup为SQL子句中的一个条件组，多个OR和AND之间需要使用条件组来组合例如 (Code like '%111%' or Name like '%2222%') and Pid='' ，最后BindParams将会拼成一个SQL子句对应模板SQL中按序出现的 %s。
```XML
<BindParams>
        <BindParamGroup ConnSymbol="and">
        			<BindParam ConnSymbol="and">
        				<FieldExpress> t.ProjectId='%s' </FieldExpress>
        				<FormName>ProjectId</FormName>
        			</BindParam>
        			<BindParam ConnSymbol="and">
                        <FieldExpress> t.CreateUserId='%s' </FieldExpress>
                        <FormName>UserId</FormName>
                    </BindParam>
        		</BindParamGroup>
	</BindParams>
	<BindParams>
            <BindParamGroup ConnSymbol="and">
            			<BindParam ConnSymbol="and">
            				<FieldExpress>  t.ProjectId='%s' </FieldExpress>
            				<FormName>ProjectId</FormName>
            			</BindParam>
            			<BindParam ConnSymbol="and">
                            <FieldExpress>  l.UserId ='%s' </FieldExpress>
                            <FormName>UserId</FormName>
                        </BindParam>
            		</BindParamGroup>
    	</BindParams>
```
