import React, { useState } from 'react'
import { Layout, Nav } from '@douyinfe/semi-ui'
import { IconExport, IconMapPin, IconHistory } from '@douyinfe/semi-icons'
import Export from './pages/Export'
import Mappings from './pages/Mappings'
import History from './pages/History'

const { Header, Sider, Content } = Layout

const items = [
  { itemKey: 'export', text: '导出', icon: <IconExport /> },
  { itemKey: 'mappings', text: '应用配置管理', icon: <IconMapPin /> },
  { itemKey: 'history', text: '导出历史', icon: <IconHistory /> }
]

function App() {
  const [activeKey, setActiveKey] = useState('export')

  const renderContent = () => {
    switch (activeKey) {
      case 'export':
        return <Export />
      case 'mappings':
        return <Mappings />
      case 'history':
        return <History />
      default:
        return <Export />
    }
  }

  return (
    <Layout style={{ height: '100vh' }}>
      <Header style={{ backgroundColor: 'var(--semi-color-bg-1)' }}>
        <div style={{ fontSize: '18px', fontWeight: 'bold', padding: '16px 24px' }}>
          导出服务
        </div>
      </Header>
      <Layout>
        <Sider style={{ backgroundColor: 'var(--semi-color-bg-1)' }}>
          <Nav
            items={items}
            selectedKeys={[activeKey]}
            onSelect={({ itemKey }) => setActiveKey(itemKey)}
            style={{ maxWidth: 220, height: '100%' }}
          />
        </Sider>
        <Content
          style={{
            padding: '24px',
            backgroundColor: 'var(--semi-color-bg-0)',
            overflow: 'auto'
          }}
        >
          {renderContent()}
        </Content>
      </Layout>
    </Layout>
  )
}

export default App
