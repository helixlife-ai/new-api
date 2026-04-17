import React, { useState, useEffect } from 'react'
import {
  Card,
  Table,
  DatePicker,
  Button,
  Toast,
  Space,
  Typography,
  Tag
} from '@douyinfe/semi-ui'
import { IconRefresh, IconDownload } from '@douyinfe/semi-icons'
import api from '../../helpers/api'
import dayjs from 'dayjs'

const { Title, Text } = Typography

function History() {
  const [data, setData] = useState([])
  const [loading, setLoading] = useState(false)
  const [timeRange, setTimeRange] = useState(null)

  const fetchData = async () => {
    setLoading(true)
    try {
      const params = {}
      if (timeRange) {
        params.startTime = timeRange.startTime
        params.endTime = timeRange.endTime
      }

      const response = await api.get('/history', { params })
      setData(response?.data || [])
    } catch (error) {
      Toast.error('获取导出历史失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleTimeRangeChange = (dates) => {
    if (dates && dates.length === 2) {
      setTimeRange({
        startTime: dayjs(dates[0]).unix(),
        endTime: dayjs(dates[1]).unix()
      })
    } else {
      setTimeRange(null)
    }
  }

  const handleSearch = () => {
    fetchData()
  }

  const handleDownload = (record) => {
    if (record.downloadUrl) {
      window.open(record.downloadUrl, '_blank')
    } else {
      Toast.warning('该记录没有可下载的文件')
    }
  }

  const getStatusTag = (status) => {
    const statusMap = {
      success: { color: 'green', text: '成功' },
      failed: { color: 'red', text: '失败' },
      processing: { color: 'blue', text: '处理中' }
    }
    const config = statusMap[status] || { color: 'default', text: status }
    return <Tag color={config.color}>{config.text}</Tag>
  }

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80
    },
    {
      title: '导出时间',
      dataIndex: 'createdAt',
      render: (val) => val ? dayjs(val).format('YYYY-MM-DD HH:mm:ss') : '-',
      sorter: (a, b) => new Date(a.createdAt) - new Date(b.createdAt)
    },
    {
      title: '操作人',
      dataIndex: 'operator'
    },
    {
      title: '时间范围',
      render: (_, record) => (
        <Text>
          {record.startTime ? dayjs.unix(record.startTime).format('MM-DD HH:mm') : '-'} ~ {' '}
          {record.endTime ? dayjs.unix(record.endTime).format('MM-DD HH:mm') : '-'}
        </Text>
      )
    },
    {
      title: '筛选条件',
      dataIndex: 'filterType',
      render: (val) => {
        const typeMap = {
          feishu_csv: '飞书 CSV',
          token_name: '令牌名称'
        }
        return <Tag>{typeMap[val] || val}</Tag>
      }
    },
    {
      title: '导出记录数',
      dataIndex: 'recordCount',
      align: 'right'
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: getStatusTag
    },
    {
      title: '操作',
      render: (_, record) => (
        <Space>
          <Button
            icon={<IconDownload />}
            size="small"
            disabled={record.status !== 'success'}
            onClick={() => handleDownload(record)}
          >
            下载
          </Button>
        </Space>
      )
    }
  ]

  return (
    <div>
      <Title heading={3} style={{ marginBottom: '24px' }}>
        导出历史
      </Title>

      <Card>
        <Space spacing="loose" style={{ marginBottom: '16px', flexWrap: 'wrap' }}>
          <div>
            <Text type="secondary" style={{ marginRight: '8px' }}>时间范围:</Text>
            <DatePicker
              type="dateTimeRange"
              onChange={handleTimeRangeChange}
              placeholder={['开始时间', '结束时间']}
              style={{ width: '320px' }}
              showClear
            />
          </div>
          <Button type="primary" onClick={handleSearch}>查询</Button>
          <Button icon={<IconRefresh />} onClick={fetchData}>刷新</Button>
        </Space>

        <Table
          columns={columns}
          dataSource={data}
          loading={loading}
          pagination={{
            pageSize: 10,
            showSizeChanger: true,
            pageSizeOpts: [10, 20, 50, 100]
          }}
          rowKey="id"
        />
      </Card>
    </div>
  )
}

export default History
