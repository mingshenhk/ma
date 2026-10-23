package hongershi

import "errors"

const OriginalID = 110
const Status = "blocked: platehes module absent in archive and all nested packages"

func Start() error {
	return errors.New("红二十仅有菜单ID110，缺失牌组和规则模块；不能开局")
}
