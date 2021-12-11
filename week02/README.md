

#### 问题: 我们在数据库操作的时候，比如dao层中遇到一个sql.ErrNoRows 的时候，是否应该Wrap这个error，抛给上层。为什么，应该怎么做请写出代码？

##### errors.Wrap()
首先, 在我们日常开发中, 经常需要为执行的报错结果, 进行处理并记录, 甚至在并发情况下, 分不清记录的顺序.
这时候 github.com/pkg/errors 便衍生 **Wrap()** 方法.

- 作用
    - 可包装底层错误, 
    - 增加上下文的文本描述信息并附加调用栈. 
    - 为我们在调试时提供极大的便利性.

##### sql.ErrNoRows
而 sql.ErrNoRows 是 database/sql 包的特殊错误常量. 当结果集为空时, QueryRow() 就会返回它.

一个空的结果集或许往往不应该被认为是应用的错误, 但我们需按业务情况的特殊性分析处理
```
func (m *Model) GetUserUser(c context.Context, uid int) (result *UserUser, err error) {
	result 				= &UserUser{}

	err 				= m.UserDb.QueryRow(c, _GetUserUser, uid).Scan(&result.Uid, &result.Appid, &result.Tel, &result.Nickname, &result.Headimgurl, &result.Sex)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}
```
比如上述dao层, 我们先通过异构索引表获得uid, 再去user表获得用户所有用户信息, 理论上是可以获得用户信息, 如果获取不到用户信息, 我们也可以通过result.Uid是否大于0判断

但如果得到的错误是非sql.ErrNoRows, 需看上层调用逻辑的具体情况, 而确定是否需加入errors.Wrap(), 做本层的错误描述
