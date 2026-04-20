import React, { useState } from 'react'
import { Input, Button, Toast, Typography, Space } from '@douyinfe/semi-ui'
import { IconSearch } from '@douyinfe/semi-icons'

const { Text } = Typography

function TokenNameSearch({ timeRange, onTokenNameChange, tokenName }) {
  const [inputValue, setInputValue] = useState(tokenName)

  const handleInputChange = (value) => {
    setInputValue(value)
  }

  const handleSearch = () => {
    if (!inputValue.trim()) {
      Toast.warning('请输入令牌名称')
      return
    }
    if (!timeRange) {
      Toast.warning('请先选择时间范围')
      return
    }
    onTokenNameChange(inputValue.trim())
  }

  const handleKeyPress = (e) => {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }

  return (
    <div>
      <Space spacing="tight">
        <Input
          value={inputValue}
          onChange={handleInputChange}
          onKeyPress={handleKeyPress}
          placeholder="输入令牌名称进行搜索"
          style={{ width: '280px' }}
          prefix={<IconSearch />}
          showClear
        />
        <Button
          type="primary"
          icon={<IconSearch />}
          onClick={handleSearch}
        >
          搜索
        </Button>
      </Space>

      <Text type="tertiary" size="small" style={{ display: 'block', marginTop: '8px' }}>
        输入至少 2 个字符后点击搜索，将自动匹配相关令牌
      </Text>
    </div>
  )
}

export default TokenNameSearch
