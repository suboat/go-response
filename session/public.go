package session

//import (
//	"github.com/suboat/sorm"
//)

//// 根据uid取出用户当前状态
//// TODO:自己维护一个高速的方法
//func PubUserGetByUid(uid orm.Uid) (u *userBase, err error) {
//	if UserModel == nil {
//		err = orm.ErrModelUndefined
//		return
//	}
//
//	u = new(userBase)
//	if err = UserModel.Objects().Filter(orm.M{"uid": uid}).One(u); err != nil {
//		// 查询失败或不唯一: 验证失败
//		return
//	}
//	return
//}
