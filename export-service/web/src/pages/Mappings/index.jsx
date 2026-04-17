import React, { useState, useEffect, useRef } from 'react'
import {
  Card,
  Table,
  Button,
  Modal,
  Form,
  Input,
  Select,
  Tag,
  Toast,
  Space,
  Popconfirm,
  Typography,
  Divider,
  Progress
} from '@douyinfe/semi-ui'
import {
  IconPlus,
  IconDelete,
  IconEdit,
  IconUpload,
  IconRefresh,
  IconSearch
} from '@douyinfe/semi-icons'
import api from '../../helpers/api'
import axios from 'axios'

const { Title, Text } = Typography

function Mappings() {
  const [data, setData] = useState([])
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingRecord, setEditingRecord] = useState(null)
  const [selectedTags, setSelectedTags] = useState([])
  const [filterFeishuAppId, setFilterFeishuAppId] = useState('')
  const [filterFeishuName, setFilterFeishuName] = useState('')
  const [filterTokenName, setFilterTokenName] = useState('')
  const [pageSize, setPageSize] = useState(10)
  const [formApi, setFormApi] = useState(null)

  // 上传队列状态
  const [uploadQueue, setUploadQueue] = useState([])
  const fileInputRef = useRef(null)

  const fetchData = async () => {
    setLoading(true)
    try {
      const response = await api.get('/mappings')
      setData(response || [])
    } catch (error) {
      Toast.error('获取映射列表失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  // 监听上传队列变化，全部成功时刷新数据并清空队列
  useEffect(() => {
    const allSuccess = uploadQueue.length > 0 &&
      uploadQueue.every(item => item.status === 'success')

    if (allSuccess) {
      const totalImported = uploadQueue.reduce((sum, item) => sum + (item.importedCount || 0), 0)
      Toast.success(`成功导入 ${totalImported} 条映射`)
      fetchData()
      setTimeout(() => setUploadQueue([]), 1500)
    }
  }, [uploadQueue])

  const allTags = [...new Set(data.flatMap(item => item.tags || []))]

  const filteredData = data.filter(item => {
    if (selectedTags.length > 0 && !selectedTags.some(tag => (item.tags || []).includes(tag))) return false
    if (filterFeishuAppId && !item.feishuAppId?.toLowerCase().includes(filterFeishuAppId.toLowerCase())) return false
    if (filterFeishuName && !item.feishuName?.toLowerCase().includes(filterFeishuName.toLowerCase())) return false
    if (filterTokenName && !item.tokenName?.toLowerCase().includes(filterTokenName.toLowerCase())) return false
    return true
  })

  const handleAdd = () => {
    setEditingRecord(null)
    setModalVisible(true)
  }

  const handleEdit = (record) => {
    setEditingRecord(record)
    setModalVisible(true)
    if (formApi) {
      formApi.setValues(record)
    }
  }

  const handleDelete = async (id) => {
    try {
      await api.delete(`/mappings/${id}`)
      Toast.success('删除成功')
      fetchData()
    } catch (error) {
      Toast.error('删除失败')
    }
  }

  const handleBatchDeleteByTag = async () => {
    if (selectedTags.length === 0) {
      Toast.warning('请先选择标签')
      return
    }

    try {
      await api.post('/mappings/batch-delete', { tags: selectedTags })
      Toast.success('批量删除成功')
      setSelectedTags([])
      fetchData()
    } catch (error) {
      Toast.error('批量删除失败')
    }
  }

  const handleSubmit = async (values) => {
    try {
      if (editingRecord) {
        await api.put(`/mappings/${editingRecord.id}`, values)
        Toast.success('更新成功')
      } else {
        await api.post('/mappings', values)
        Toast.success('创建成功')
      }
      setModalVisible(false)
      fetchData()
    } catch (error) {
      Toast.error(editingRecord ? '更新失败' : '创建失败')
    }
  }

  const handleFileInputChange = async (e) => {
    const files = Array.from(e.target.files)
    if (!files.length) return

    for (let index = 0; index < files.length; index++) {
      const file = files[index]
      const newUploadItem = {
        id: Date.now() + index,
        file,
        name: file.name,
        progress: 0,
        status: 'idle'
      }
      setUploadQueue(prev => [...prev, newUploadItem])
      setTimeout(() => startUpload(newUploadItem.id, file), 0)
    }

    // 重置 input，允许重复选择同一文件
    e.target.value = ''
  }

  const startUpload = async (uploadId, file) => {
    // 更新状态为上传中
    setUploadQueue(prev => prev.map(item =>
      item.id === uploadId ? { ...item, status: 'uploading', progress: 0 } : item
    ))

    const formData = new FormData()
    formData.append('file', file) // 直接传递 File 对象

    try {
      const response = await axios.post('/api/mappings/import', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        },
        onUploadProgress: (progressEvent) => {
          const percentCompleted = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          )
          setUploadQueue(prev => prev.map(item =>
            item.id === uploadId ? { ...item, progress: percentCompleted } : item
          ))
        }
      })

      // 上传成功
      setUploadQueue(prev => prev.map(item =>
        item.id === uploadId
          ? { ...item, status: 'success', progress: 100, importedCount: response.data?.imported_count || 0 }
          : item
      ))
    } catch (error) {
      const errorMsg = error.response?.data?.error || error.message || '上传失败'
      setUploadQueue(prev => prev.map(item =>
        item.id === uploadId
          ? { ...item, status: 'error', error: errorMsg }
          : item
      ))
    }
  }

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80
    },
    {
      title: '飞书 App ID',
      dataIndex: 'feishuAppId'
    },
    {
      title: '飞书名称',
      dataIndex: 'feishuName'
    },
    {
      title: 'Token ID',
      dataIndex: 'tokenId',
      width: 100
    },
    {
      title: '令牌名称',
      dataIndex: 'tokenName'
    },
    {
      title: '标签',
      dataIndex: 'tags',
      render: (tags) => (
        <Space>
          {(tags || []).map((tag, index) => (
            <Tag key={index} color="blue" size="small">{tag}</Tag>
          ))}
        </Space>
      )
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      render: (val) => val ? new Date(val).toLocaleString() : '-'
    },
    {
      title: '操作',
      render: (_, record) => (
        <Space>
          <Button
            icon={<IconEdit />}
            size="small"
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确认删除"
            content="确定要删除这条映射吗？"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button icon={<IconDelete />} size="small" type="danger">删除</Button>
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <div>
      <Title heading={3} style={{ marginBottom: '24px' }}>
        应用配置管理
      </Title>

      <Card>
        <Space spacing="tight" style={{ marginBottom: '16px', flexWrap: 'wrap', alignItems: 'flex-start' }}>
          <Button type="primary" icon={<IconPlus />} onClick={handleAdd}>
            新增映射
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            accept=".csv"
            multiple
            style={{ display: 'none' }}
            onChange={handleFileInputChange}
          />
          <Button icon={<IconUpload />} onClick={() => fileInputRef.current.click()}>导入 CSV</Button>
          <Button icon={<IconRefresh />} onClick={fetchData}>刷新</Button>
        </Space>

        <Divider />

        <div style={{ marginBottom: '16px', display: 'flex', flexWrap: 'wrap', gap: '8px', alignItems: 'center' }}>
          <Input
            prefix={<IconSearch />}
            placeholder="飞书 App ID"
            value={filterFeishuAppId}
            onChange={v => setFilterFeishuAppId(v)}
            style={{ width: 180 }}
          />
          <Input
            prefix={<IconSearch />}
            placeholder="飞书名称"
            value={filterFeishuName}
            onChange={v => setFilterFeishuName(v)}
            style={{ width: 160 }}
          />
          <Input
            prefix={<IconSearch />}
            placeholder="令牌名称"
            value={filterTokenName}
            onChange={v => setFilterTokenName(v)}
            style={{ width: 160 }}
          />
          <Text type="secondary" style={{ marginRight: '4px' }}>标签:</Text>
          <Select
            multiple
            value={selectedTags}
            onChange={setSelectedTags}
            placeholder="选择标签进行筛选"
            style={{ width: '300px', marginRight: '8px' }}
          >
            {allTags.map(tag => (
              <Select.Option key={tag} value={tag}>{tag}</Select.Option>
            ))}
          </Select>
          {selectedTags.length > 0 && (
            <Popconfirm
              title="批量删除"
              content={`确定要删除标签为 "${selectedTags.join(', ')}" 的所有映射吗？`}
              onConfirm={handleBatchDeleteByTag}
            >
              <Button type="danger" icon={<IconDelete />}>批量删除</Button>
            </Popconfirm>
          )}
        </div>

        <Table
          columns={columns}
          dataSource={filteredData}
          loading={loading}
          pagination={{
            pageSize,
            pageSizeOpts: [10, 20, 50, 100],
            showSizeChanger: true,
            onPageSizeChange: size => setPageSize(size)
          }}
          rowKey="id"
        />
      </Card>

      <Modal
        title={editingRecord ? '编辑映射' : '新增映射'}
        visible={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
      >
        <Form
          getFormApi={setFormApi}
          initValues={editingRecord || {}}
          onSubmit={handleSubmit}
        >
          {({ formState, values, formApi }) => (
            <>
              <Form.Input
                field="feishuAppId"
                label="飞书 App ID"
                placeholder="请输入飞书 App ID"
                rules={[{ required: true, message: '请输入飞书 App ID' }]}
              />
              <Form.Input
                field="feishuName"
                label="飞书名称"
                placeholder="请输入飞书名称"
                rules={[{ required: true, message: '请输入飞书名称' }]}
              />
              <Form.InputNumber
                field="tokenId"
                label="Token ID"
                placeholder="请输入 Token ID"
                rules={[{ required: true, message: '请输入 Token ID' }]}
                style={{ width: '100%' }}
              />
              <Form.Input
                field="tokenName"
                label="令牌名称"
                placeholder="请输入令牌名称"
                rules={[{ required: true, message: '请输入令牌名称' }]}
              />
              <Form.TagInput
                field="tags"
                label="标签"
                placeholder="输入标签后按回车"
                allowDuplicates={false}
              />
              <div style={{ textAlign: 'right', marginTop: '16px' }}>
                <Space>
                  <Button onClick={() => setModalVisible(false)}>取消</Button>
                  <Button type="primary" htmlType="submit">保存</Button>
                </Space>
              </div>
            </>
          )}
        </Form>
      </Modal>

      {/* 上传进度浮层：直接由 React 渲染，原地更新，避免 Toast 关闭/重开导致的多条堆叠问题 */}
      {uploadQueue.length > 0 && (
        <div style={{
          position: 'fixed',
          bottom: 24,
          right: 24,
          zIndex: 1000,
          background: 'var(--semi-color-bg-2)',
          borderRadius: 8,
          boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
          padding: '12px 16px',
          minWidth: 280,
        }}>
          {uploadQueue.map((item) => (
            <div key={item.id} style={{ marginBottom: '12px' }}>
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '6px' }}>
                <span style={{
                  flex: 1, overflow: 'hidden', textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap', marginRight: 8, fontSize: 13
                }}>
                  {item.name}
                </span>
                <span style={{
                  fontSize: 12, whiteSpace: 'nowrap',
                  color: item.status === 'error' ? '#ff4d4f'
                       : item.status === 'success' ? '#52c41a'
                       : item.status === 'uploading' ? '#1890ff' : '#666'
                }}>
                  {item.status === 'uploading' && '上传中...'}
                  {item.status === 'success' && '上传成功'}
                  {item.status === 'error' && '上传失败'}
                  {item.status === 'idle' && '等待上传'}
                </span>
              </div>
              {item.status === 'uploading' && (
                <>
                  <Progress percent={item.progress} size="small" showInfo={false} />
                  <div style={{ textAlign: 'right', fontSize: 12, marginTop: 2 }}>
                    {item.progress}%
                  </div>
                </>
              )}
              {item.status === 'error' && item.error && (
                <div style={{ color: '#ff4d4f', fontSize: 12 }}>{item.error}</div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export default Mappings
