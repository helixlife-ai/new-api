import React, { useState } from 'react'
import { Upload, Toast, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconUpload } from '@douyinfe/semi-icons'
import Papa from 'papaparse'

const { Text } = Typography

const MAX_ROWS = 200

function FeishuCsvUpload({ timeRange, onParsed, appIds }) {
  const [fileList, setFileList] = useState([])
  const [parsedData, setParsedData] = useState({
    totalRows: 0,
    fileName: ''
  })

  const handleFileChange = ({ fileList: newFileList }) => {
    setFileList(newFileList)

    if (newFileList.length === 0) {
      setParsedData({ totalRows: 0, fileName: '' })
      onParsed([])
    }
  }

  const beforeUpload = ({ file }) => {
    if (!timeRange) {
      Toast.warning('请先选择时间范围')
      return false
    }

    if (!file.name.endsWith('.csv')) {
      Toast.error('请上传 CSV 文件')
      return false
    }

    const rawFile = file.fileInstance
    if (!rawFile) {
      Toast.error('无法读取文件，请重试')
      return false
    }

    const reader = new FileReader()
    reader.onload = (e) => {
      const content = e.target.result

      Papa.parse(content, {
        header: true,
        skipEmptyLines: true,
        complete: (results) => {
          const rows = results.data

          if (rows.length > MAX_ROWS) {
            Toast.error(`CSV 文件行数超过限制，最多支持 ${MAX_ROWS} 行`)
            setFileList([])
            return
          }

          const appIds = []
          rows.forEach((row, index) => {
            const appId = row.feishu_app_id || row['feishu_app_id'] || row.app_id || row['app_id']
            if (appId && appId.trim()) {
              appIds.push({
                row: index + 2,
                value: appId.trim()
              })
            }
          })

          const uniqueAppIds = [...new Set(appIds.map(item => item.value))]

          setParsedData({
            totalRows: rows.length,
            fileName: file.name
          })

          onParsed(uniqueAppIds)
          Toast.success(`成功解析 ${rows.length} 行数据，提取 ${uniqueAppIds.length} 个唯一 App ID`)
        },
        error: (error) => {
          Toast.error(`CSV 解析失败: ${error.message}`)
          setFileList([])
        }
      })
    }

    reader.onerror = () => {
      Toast.error('文件读取失败')
      setFileList([])
    }

    reader.readAsText(rawFile)
    return false
  }

  return (
    <div>
      <Upload
        fileList={fileList}
        onChange={handleFileChange}
        beforeUpload={beforeUpload}
        accept=".csv"
        maxSize={1024}
        showUploadList={true}
        draggable
        style={{ width: '100%' }}
      >
        <div
          style={{
            width: '100%',
            height: '100px',
            border: '2px dashed var(--semi-color-border)',
            borderRadius: '4px',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            cursor: 'pointer',
            backgroundColor: fileList.length > 0 ? 'var(--semi-color-fill-0)' : 'transparent'
          }}
        >
          <IconUpload size="large" style={{ marginBottom: '8px', color: 'var(--semi-color-text-2)' }} />
          <Text>点击或拖拽上传 CSV 文件</Text>
          <Text type="tertiary" size="small">支持包含 feishu_app_id 列的 CSV，最多 {MAX_ROWS} 行</Text>
        </div>
      </Upload>

      {appIds.length > 0 && (
        <div style={{ marginTop: '12px', padding: '12px', backgroundColor: 'var(--semi-color-fill-0)', borderRadius: '4px' }}>
          <Space vertical spacing="tight" style={{ width: '100%' }}>
            <div>
              <Text type="secondary" size="small">文件名: </Text>
              <Text strong size="small">{parsedData.fileName}</Text>
            </div>
            <div>
              <Text type="secondary" size="small">总行数: </Text>
              <Text strong size="small">{parsedData.totalRows}</Text>
            </div>
            <div>
              <Text type="secondary" size="small">提取 App ID: </Text>
              <Tag color="blue" size="small">{appIds.length} 个</Tag>
            </div>
            <div style={{ marginTop: '4px' }}>
              <Text type="secondary" size="small">预览 (前 10 个):</Text>
              <div style={{ marginTop: '4px' }}>
                {appIds.slice(0, 10).map((id, index) => (
                  <Tag key={index} size="small" style={{ margin: '2px' }}>{id}</Tag>
                ))}
                {appIds.length > 10 && (
                  <Tag size="small" style={{ margin: '2px' }}>+{appIds.length - 10}</Tag>
                )}
              </div>
            </div>
          </Space>
        </div>
      )}
    </div>
  )
}

export default FeishuCsvUpload
