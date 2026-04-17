package model

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// FeishuTokenMapping 飞书应用与Token的映射关系
type FeishuTokenMapping struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	FeishuAppID string    `gorm:"uniqueIndex;size:64" json:"feishu_app_id"`
	FeishuName  string    `gorm:"size:128" json:"feishu_name"`
	TokenID     int       `gorm:"uniqueIndex" json:"token_id"`
	TokenName   string    `gorm:"size:128" json:"token_name"`
	Labels      string    `gorm:"size:256" json:"labels"` // 批次标签
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TableName 指定表名
func (FeishuTokenMapping) TableName() string {
	return "feishu_token_mappings"
}

// GetMappingByFeishuAppID 根据飞书AppID获取映射
func GetMappingByFeishuAppID(appID string) (*FeishuTokenMapping, error) {
	var mapping FeishuTokenMapping
	if err := DB.Where("feishu_app_id = ?", appID).First(&mapping).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

// GetMappingByTokenID 根据TokenID获取映射
func GetMappingByTokenID(tokenID int) (*FeishuTokenMapping, error) {
	var mapping FeishuTokenMapping
	if err := DB.Where("token_id = ?", tokenID).First(&mapping).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

// GetMappingsByLabel 根据标签获取映射列表
func GetMappingsByLabel(label string) ([]FeishuTokenMapping, error) {
	var mappings []FeishuTokenMapping
	// 使用 LIKE 查询支持多个标签或部分匹配
	if err := DB.Where("labels LIKE ?", "%"+label+"%").Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

// GetAllMappings 获取所有映射
func GetAllMappings() ([]FeishuTokenMapping, error) {
	var mappings []FeishuTokenMapping
	if err := DB.Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

// GetMappingsByLabels 根据多个标签获取映射（OR关系）
func GetMappingsByLabels(labels []string) ([]FeishuTokenMapping, error) {
	if len(labels) == 0 {
		return GetAllMappings()
	}

	var mappings []FeishuTokenMapping
	query := DB
	for i, label := range labels {
		if i == 0 {
			query = query.Where("labels LIKE ?", "%"+label+"%")
		} else {
			query = query.Or("labels LIKE ?", "%"+label+"%")
		}
	}
	if err := query.Find(&mappings).Error; err != nil {
		return nil, err
	}
	return mappings, nil
}

// GetMappingsByLabelExact 精确匹配标签（非LIKE模糊匹配）
// 读取所有记录，按逗号分割labels后精确匹配
func GetMappingsByLabelExact(label string) ([]FeishuTokenMapping, error) {
	if label == "" {
		return GetAllMappings()
	}

	allMappings, err := GetAllMappings()
	if err != nil {
		return nil, err
	}

	var result []FeishuTokenMapping
	for _, mapping := range allMappings {
		if mapping.Labels == "" {
			continue
		}
		// 按逗号分割labels并精确匹配
		labels := strings.Split(mapping.Labels, ",")
		for _, l := range labels {
			if strings.TrimSpace(l) == label {
				result = append(result, mapping)
				break
			}
		}
	}
	return result, nil
}

// GetMappingsByLabelsExact 根据多个标签精确匹配（OR关系）
func GetMappingsByLabelsExact(labels []string) ([]FeishuTokenMapping, error) {
	if len(labels) == 0 {
		return GetAllMappings()
	}

	allMappings, err := GetAllMappings()
	if err != nil {
		return nil, err
	}

	var result []FeishuTokenMapping
	for _, mapping := range allMappings {
		if mapping.Labels == "" {
			continue
		}
		// 按逗号分割labels
		mappingLabels := strings.Split(mapping.Labels, ",")
		// 检查是否有任一标签匹配
		for _, ml := range mappingLabels {
			ml = strings.TrimSpace(ml)
			for _, targetLabel := range labels {
				if ml == targetLabel {
					result = append(result, mapping)
					break
				}
			}
		}
	}
	return result, nil
}

// DeleteMappingsByLabelExact 根据标签精确匹配删除（非LIKE模糊匹配）
func DeleteMappingsByLabelExact(label string) error {
	mappings, err := GetMappingsByLabelExact(label)
	if err != nil {
		return err
	}

	if len(mappings) == 0 {
		return nil
	}

	ids := make([]uint, len(mappings))
	for i, m := range mappings {
		ids[i] = m.ID
	}
	return DeleteMappingsByIDs(ids)
}

// DeleteMappingsByLabelsExact 根据多个标签精确匹配删除（OR关系）
func DeleteMappingsByLabelsExact(labels []string) (int64, error) {
	if len(labels) == 0 {
		return 0, nil
	}

	mappings, err := GetMappingsByLabelsExact(labels)
	if err != nil {
		return 0, err
	}

	if len(mappings) == 0 {
		return 0, nil
	}

	ids := make([]uint, len(mappings))
	for i, m := range mappings {
		ids[i] = m.ID
	}

	result := DB.Where("id IN ?", ids).Delete(&FeishuTokenMapping{})
	return result.RowsAffected, result.Error
}

// CreateMapping 创建映射
func CreateMapping(mapping *FeishuTokenMapping) error {
	mapping.CreatedAt = time.Now()
	mapping.UpdatedAt = time.Now()
	return DB.Create(mapping).Error
}

// UpdateMapping 更新映射
func UpdateMapping(id uint, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	return DB.Model(&FeishuTokenMapping{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteMapping 删除单条映射
func DeleteMapping(id uint) error {
	return DB.Delete(&FeishuTokenMapping{}, id).Error
}

// DeleteMappingsByLabel 根据标签批量删除
func DeleteMappingsByLabel(label string) error {
	return DB.Where("labels LIKE ?", "%"+label+"%").Delete(&FeishuTokenMapping{}).Error
}

// DeleteMappingsByIDs 根据ID批量删除
func DeleteMappingsByIDs(ids []uint) error {
	return DB.Where("id IN ?", ids).Delete(&FeishuTokenMapping{}).Error
}

// GetMappingByID 根据ID获取映射
func GetMappingByID(id uint) (*FeishuTokenMapping, error) {
	var mapping FeishuTokenMapping
	if err := DB.First(&mapping, id).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

// toUTF8Reader 检测并转换 CSV 字节流的编码为 UTF-8。
// 处理顺序：
//  1. UTF-8 BOM → 剥除 BOM，原样使用
//  2. 有效 UTF-8 → 原样使用
//  3. 其他 → 按 GBK 解码（Windows Excel 默认编码）
func toUTF8Reader(r io.Reader) (io.Reader, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// 剥除 UTF-8 BOM
	if bytes.HasPrefix(raw, []byte{0xEF, 0xBB, 0xBF}) {
		raw = raw[3:]
	}

	if utf8.Valid(raw) {
		return bytes.NewReader(raw), nil
	}

	// 尝试 GBK 解码
	decoded, _, err := transform.Bytes(simplifiedchinese.GBK.NewDecoder(), raw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file as GBK: %w", err)
	}
	return bytes.NewReader(decoded), nil
}

// ImportMappingsFromCSV 从CSV导入映射
// CSV格式: feishu_app_id,feishu_name,token_id,token_name,labels(可选)
// overrideLabels: 若非空，则覆盖所有导入行的 labels 字段（忽略 CSV 中的 labels 列）
func ImportMappingsFromCSV(reader io.Reader, overrideLabels string) (int, error) {
	utf8Reader, err := toUTF8Reader(reader)
	if err != nil {
		return 0, fmt.Errorf("failed to read file: %w", err)
	}
	csvReader := csv.NewReader(utf8Reader)
	csvReader.FieldsPerRecord = -1 // 允许可变字段数

	// 读取表头
	headers, err := csvReader.Read()
	if err != nil {
		return 0, fmt.Errorf("failed to read CSV header: %w", err)
	}

	// 验证表头
	if len(headers) < 4 {
		return 0, fmt.Errorf("CSV must have at least 4 columns: feishu_app_id, feishu_name, token_id, token_name")
	}

	// 标准化表头
	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(strings.ToLower(h))] = i
	}

	requiredCols := []string{"feishu_app_id", "feishu_name", "token_id", "token_name"}
	for _, col := range requiredCols {
		if _, ok := headerMap[col]; !ok {
			return 0, fmt.Errorf("missing required column: %s", col)
		}
	}

	var importedCount int
	var lineNum int = 1

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return importedCount, fmt.Errorf("error reading line %d: %w", lineNum, err)
		}
		lineNum++

		// 跳过空行
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}

		// 解析token_id
		tokenID, err := strconv.Atoi(strings.TrimSpace(record[headerMap["token_id"]]))
		if err != nil {
			return importedCount, fmt.Errorf("invalid token_id at line %d: %w", lineNum, err)
		}

		mapping := &FeishuTokenMapping{
			FeishuAppID: strings.TrimSpace(record[headerMap["feishu_app_id"]]),
			FeishuName:  strings.TrimSpace(record[headerMap["feishu_name"]]),
			TokenID:     tokenID,
			TokenName:   strings.TrimSpace(record[headerMap["token_name"]]),
		}

		// labels 优先级：overrideLabels（表单参数）> CSV labels 列
		if overrideLabels != "" {
			mapping.Labels = overrideLabels
		} else if labelsIdx, ok := headerMap["labels"]; ok && labelsIdx < len(record) {
			mapping.Labels = strings.TrimSpace(record[labelsIdx])
		}

		// 检查是否已存在
		existing, _ := GetMappingByFeishuAppID(mapping.FeishuAppID)
		if existing != nil {
			// 更新现有记录
			updates := map[string]interface{}{
				"feishu_name": mapping.FeishuName,
				"token_id":    mapping.TokenID,
				"token_name":  mapping.TokenName,
				"labels":      mapping.Labels,
			}
			if err := UpdateMapping(existing.ID, updates); err != nil {
				return importedCount, fmt.Errorf("failed to update mapping at line %d: %w", lineNum, err)
			}
		} else {
			// 创建新记录
			if err := CreateMapping(mapping); err != nil {
				return importedCount, fmt.Errorf("failed to create mapping at line %d: %w", lineNum, err)
			}
		}

		importedCount++
	}

	return importedCount, nil
}
