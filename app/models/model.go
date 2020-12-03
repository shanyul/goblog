package models

import "goblog/pkg/types"

// BaseModel 模型基类
type BaseModel struct{
	ID uint64
}

// GetStringID 获取 ID 的字符串格式
func (a BaseModel) GetStringID() string  {
	return types.UInt64ToString(a.ID)
}

