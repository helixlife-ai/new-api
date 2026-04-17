import React, { useState, useCallback } from 'react'
import { Card, Radio, Typography, Row, Col, Space, Tag, Badge } from '@douyinfe/semi-ui'
import { IconCalendar, IconUpload, IconSearch, IconDownload } from '@douyinfe/semi-icons'
import TimeRangePicker from './components/TimeRangePicker'
import FeishuCsvUpload from './components/FeishuCsvUpload'
import TokenNameSearch from './components/TokenNameSearch'
import ExportActions from './components/ExportActions'
import api from '../../helpers/api'

const { Title, Text } = Typography

const FILTER_MODE = {
  FEISHU_CSV: 'feishu_csv',
  TOKEN_NAME: 'token_name'
}

// 步骤条组件
function StepItem({ number, title, active, children }) {
  return (
    <div style={{ display: 'flex', gap: '16px', marginBottom: '24px' }}>
      <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        <Badge
          count={number}
          style={{
            backgroundColor: active ? 'var(--semi-color-primary)' : 'var(--semi-color-fill-2)',
            color: active ? 'white' : 'var(--semi-color-text-2)',
            width: '28px',
            height: '28px',
            borderRadius: '50%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '14px',
            fontWeight: 'bold'
          }}
        />
        <div
          style={{
            width: '2px',
            flex: 1,
            backgroundColor: 'var(--semi-color-border)',
            marginTop: '8px',
            minHeight: '20px'
          }}
        />
      </div>
      <div style={{ flex: 1, paddingTop: '2px' }}>
        <Text strong style={{ fontSize: '16px', display: 'block', marginBottom: '12px' }}>
          {title}
        </Text>
        {children}
      </div>
    </div>
  )
}

function Export() {
  const [timeRange, setTimeRange] = useState(null)
  const [filterMode, setFilterMode] = useState(FILTER_MODE.FEISHU_CSV)
  const [feishuAppIds, setFeishuAppIds] = useState([])
  const [tokenName, setTokenName] = useState('')
  const [previewData, setPreviewData] = useState(null)
  const [previewLoading, setPreviewLoading] = useState(false)

  const handleTimeRangeChange = (range) => {
    setTimeRange(range)
    setPreviewData(null)
  }

  const handleFeishuCsvParsed = useCallback(async (appIds) => {
    setFeishuAppIds(appIds)
    setPreviewData(null)

    if (appIds.length > 0 && timeRange) {
      await fetchPreview({ filter_type: 'feishu_csv', feishu_app_ids: appIds })
    }
  }, [timeRange])

  const handleTokenNameChange = useCallback(async (name) => {
    setTokenName(name)
    setPreviewData(null)

    if (name && name.length >= 2 && timeRange) {
      await fetchPreview({ filter_type: 'token_name', token_name: name })
    }
  }, [timeRange])

  const fetchPreview = async (params) => {
    if (!timeRange) return

    setPreviewLoading(true)
    try {
      const response = await api.post('/export/preview', {
        ...params,
        time_range_start: timeRange.startTime,
        time_range_end: timeRange.endTime
      })
      setPreviewData(response)
    } catch (error) {
      setPreviewData(null)
    } finally {
      setPreviewLoading(false)
    }
  }

  const getExportParams = () => {
    const params = {
      filter_type: filterMode,
      time_range_start: timeRange?.startTime,
      time_range_end: timeRange?.endTime
    }

    if (filterMode === FILTER_MODE.FEISHU_CSV) {
      params.feishu_app_ids = feishuAppIds
    } else {
      params.token_name = tokenName
    }

    return params
  }

  const isExportDisabled = () => {
    if (!timeRange) return true
    if (filterMode === FILTER_MODE.FEISHU_CSV) return feishuAppIds.length === 0
    return !tokenName
  }

  return (
    <div style={{ maxWidth: '1000px' }}>
      <Title heading={3} style={{ marginBottom: '24px' }}>
        导出使用记录
      </Title>

      <Row gutter={[24, 24]}>
        {/* 左侧：步骤配置区 */}
        <Col span={16}>
          <Card>
            {/* 步骤1：时间范围 */}
            <StepItem number={1} title="选择时间范围" active={!timeRange}>
              <TimeRangePicker onChange={handleTimeRangeChange} />
              {timeRange && (
                <div style={{ marginTop: '12px' }}>
                  <Tag color="green" size="small">
                    <IconCalendar style={{ marginRight: '4px' }} />
                    {timeRange.startTimeFormatted} ~ {timeRange.endTimeFormatted}
                  </Tag>
                </div>
              )}
            </StepItem>

            {/* 步骤2：筛选模式 */}
            <StepItem number={2} title="选择筛选模式" active={timeRange && !previewData}>
              <Radio.Group
                type="button"
                value={filterMode}
                onChange={(e) => {
                  setFilterMode(e.target.value)
                  setPreviewData(null)
                  setFeishuAppIds([])
                  setTokenName('')
                }}
              >
                <Radio value={FILTER_MODE.FEISHU_CSV}>
                  <Space>
                    <IconUpload />
                    上传飞书 CSV
                  </Space>
                </Radio>
                <Radio value={FILTER_MODE.TOKEN_NAME} disabled>
                  <Space>
                    <IconSearch />
                    按令牌名称搜索
                  </Space>
                </Radio>
              </Radio.Group>

              <div style={{ marginTop: '12px' }}>
                <Text type="tertiary" size="small">
                  {filterMode === FILTER_MODE.FEISHU_CSV
                    ? '上传包含飞书 App ID 的 CSV 文件，系统将匹配对应的 Token'
                    : '输入 Token 名称关键词，系统将搜索匹配的令牌'}
                </Text>
              </div>
            </StepItem>

            {/* 步骤3：筛选条件 */}
            <StepItem number={3} title="设置筛选条件" active={timeRange && (feishuAppIds.length > 0 || tokenName)}>
              {filterMode === FILTER_MODE.FEISHU_CSV ? (
                <FeishuCsvUpload
                  timeRange={timeRange}
                  onParsed={handleFeishuCsvParsed}
                  appIds={feishuAppIds}
                />
              ) : (
                <TokenNameSearch
                  timeRange={timeRange}
                  onTokenNameChange={handleTokenNameChange}
                  tokenName={tokenName}
                />
              )}
            </StepItem>

            {/* 步骤4：导出 */}
            <div style={{ display: 'flex', gap: '16px' }}>
              <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                <Badge
                  count={4}
                  style={{
                    backgroundColor: !isExportDisabled() ? 'var(--semi-color-primary)' : 'var(--semi-color-fill-2)',
                    color: !isExportDisabled() ? 'white' : 'var(--semi-color-text-2)',
                    width: '28px',
                    height: '28px',
                    borderRadius: '50%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    fontSize: '14px',
                    fontWeight: 'bold'
                  }}
                />
              </div>
              <div style={{ flex: 1, paddingTop: '2px' }}>
                <Text strong style={{ fontSize: '16px', display: 'block', marginBottom: '12px' }}>
                  导出数据
                </Text>
                <Text type="tertiary" size="small">
                  完成上述步骤后，在右侧面板点击「导出 CSV」按钮下载数据
                </Text>
              </div>
            </div>
          </Card>
        </Col>

        {/* 右侧：预览与操作区 */}
        <Col span={8}>
          <Card
            title={
              <Space>
                <IconDownload />
                导出预览
              </Space>
            }
            style={{ position: 'sticky', top: '24px' }}
          >
            <Space vertical spacing="loose" style={{ width: '100%' }}>
              {/* 配置摘要 */}
              <div>
                <Text type="secondary" style={{ display: 'block', marginBottom: '8px' }}>
                  导出配置
                </Text>
                <div style={{ padding: '12px', backgroundColor: 'var(--semi-color-fill-0)', borderRadius: '4px' }}>
                  <Space vertical spacing="tight" style={{ width: '100%' }}>
                    <div>
                      <Text type="secondary" size="small">时间范围: </Text>
                      <br />
                      <Text strong size="small">
                        {timeRange
                          ? `${timeRange.startTimeFormatted} ~ ${timeRange.endTimeFormatted}`
                          : '未选择'
                        }
                      </Text>
                    </div>
                    <div style={{ height: '1px', backgroundColor: 'var(--semi-color-border)', margin: '4px 0' }} />
                    <div>
                      <Text type="secondary" size="small">筛选模式: </Text>
                      <br />
                      <Text strong size="small">
                        {filterMode === FILTER_MODE.FEISHU_CSV ? '上传飞书 CSV' : '按令牌名称搜索'}
                      </Text>
                    </div>
                    <div style={{ height: '1px', backgroundColor: 'var(--semi-color-border)', margin: '4px 0' }} />
                    <div>
                      <Text type="secondary" size="small">筛选条件: </Text>
                      <br />
                      <Text strong size="small">
                        {filterMode === FILTER_MODE.FEISHU_CSV
                          ? feishuAppIds.length > 0 ? `${feishuAppIds.length} 个 App ID` : '未上传'
                          : tokenName || '未输入'
                        }
                      </Text>
                    </div>
                  </Space>
                </div>
              </div>

              {/* 数据预览 */}
              <div>
                <Text type="secondary" style={{ display: 'block', marginBottom: '8px' }}>
                  数据预览
                </Text>
                {previewLoading ? (
                  <div style={{ padding: '20px', textAlign: 'center', backgroundColor: 'var(--semi-color-fill-0)', borderRadius: '4px' }}>
                    <Text type="secondary">正在估算...</Text>
                  </div>
                ) : previewData ? (
                  <div style={{ padding: '12px', backgroundColor: 'var(--semi-color-fill-0)', borderRadius: '4px' }}>
                    <Space vertical spacing="tight" style={{ width: '100%' }}>
                      <div>
                        <Text type="secondary" size="small">匹配令牌: </Text>
                        <Tag color="blue" size="small">{previewData.token_count || 0}</Tag>
                      </div>
                    </Space>
                  </div>
                ) : (
                  <div style={{ padding: '20px', textAlign: 'center', backgroundColor: 'var(--semi-color-fill-0)', borderRadius: '4px' }}>
                    <Text type="secondary" size="small">
                      {isExportDisabled()
                        ? '请完成左侧步骤'
                        : '点击搜索或上传查看预览'
                      }
                    </Text>
                  </div>
                )}
              </div>

              <div style={{ height: '1px', backgroundColor: 'var(--semi-color-border)', margin: '4px 0' }} />

              {/* 导出操作 */}
              <ExportActions
                exportParams={getExportParams()}
                filterMode={filterMode}
                previewCount={previewData?.row_estimate || 0}
                disabled={isExportDisabled()}
              />
            </Space>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Export
