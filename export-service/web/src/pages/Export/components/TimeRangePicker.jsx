import React, { useState } from 'react'
import { DatePicker } from '@douyinfe/semi-ui'
import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'

dayjs.extend(utc)
dayjs.extend(timezone)

const TIMEZONE = 'Asia/Shanghai'

function TimeRangePicker({ onChange }) {
  const [dateRange, setDateRange] = useState(null)

  const handleChange = (dates) => {
    setDateRange(dates)

    if (dates && dates.length === 2) {
      const startTime = dayjs(dates[0]).tz(TIMEZONE).startOf('minute').unix()
      const endTime = dayjs(dates[1]).tz(TIMEZONE).endOf('minute').unix()

      onChange({
        startTime,
        endTime,
        startTimeFormatted: dayjs(dates[0]).tz(TIMEZONE).format('YYYY-MM-DD HH:mm:ss'),
        endTimeFormatted: dayjs(dates[1]).tz(TIMEZONE).format('YYYY-MM-DD HH:mm:ss')
      })
    } else {
      onChange(null)
    }
  }

  return (
    <DatePicker
      type="dateTimeRange"
      value={dateRange}
      onChange={handleChange}
      placeholder={['开始时间', '结束时间']}
      style={{ width: '360px' }}
      showClear
      needConfirm
    />
  )
}

export default TimeRangePicker
