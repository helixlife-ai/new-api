import React, { useState } from 'react'
import { Button, Toast, Space, Typography, Popover } from '@douyinfe/semi-ui'
import { IconDownload, IconRefresh, IconHelpCircle } from '@douyinfe/semi-icons'
import axios from 'axios'
import dayjs from 'dayjs'

const { Text, Paragraph } = Typography

function ExportActions({ exportParams, filterMode, previewCount, disabled }) {
  const [loading, setLoading] = useState(false)

  const handleExport = async () => {
    if (disabled) {
      Toast.warning('请完善导出条件')
      return
    }

    setLoading(true)

    try {
      const response = await axios({
        method: 'POST',
        url: '/api/export/download',
        data: exportParams,
        responseType: 'blob',
        timeout: 120000
      })

      const blob = new Blob([response.data], { type: 'text/csv;charset=utf-8;' })
      const link = document.createElement('a')
      const url = URL.createObjectURL(blob)

      const dateStr = dayjs().format('YYYYMMDD_HHmmss')
      const fileName = `usage_export_${dateStr}.csv`

      link.href = url
      link.setAttribute('download', fileName)
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)

      Toast.success('导出成功')
    } catch (error) {
      let errorMsg = '导出失败'
      if (error.response?.data) {
        const reader = new FileReader()
        reader.onload = () => {
          try {
            const errorData = JSON.parse(reader.result)
            errorMsg = errorData.message || errorMsg
          } catch {
            errorMsg = reader.result || errorMsg
          }
          Toast.error(errorMsg)
        }
        reader.readAsText(error.response.data)
      } else {
        Toast.error(errorMsg)
      }
    } finally {
      setLoading(false)
    }
  }

  const getHelpContent = () => {
    return (
      <div style={{ padding: '12px', maxWidth: '280px' }}>
        <Paragraph>
          <Text strong>导出说明：</Text>
        </Paragraph>
        <ul style={{ margin: '8px 0', paddingLeft: '16px' }}>
          <li>导出文件为 UTF-8 编码的 CSV 格式</li>
          <li>建议使用 Excel 或文本编辑器打开</li>
          <li>大量数据导出可能需要较长时间，请耐心等待</li>
          <li>导出完成后可在「导出历史」查看记录</li>
        </ul>
      </div>
    )
  }

  return (
    <div>
      <Space vertical spacing="loose" style={{ width: '100%' }}>
        <Button
          type="primary"
          size="large"
          icon={<IconDownload />}
          loading={loading}
          disabled={disabled}
          onClick={handleExport}
          block
        >
          导出 CSV
        </Button>

        <Space spacing="tight" style={{ width: '100%', justifyContent: 'center' }}>
          <Button
            icon={<IconRefresh />}
            disabled={loading}
            onClick={() => window.location.reload()}
            type="tertiary"
          >
            重置
          </Button>

          <Popover
            content={getHelpContent()}
            position="bottom"
            showArrow
          >
            <Button
              icon={<IconHelpCircle />}
              type="tertiary"
              theme="borderless"
            >
              帮助
            </Button>
          </Popover>
        </Space>
      </Space>
    </div>
  )
}

export default ExportActions