import { useIntl } from 'react-intl'
// @ts-ignore
import { time } from 'units-converter'
import OverlayTrigger from 'react-bootstrap/OverlayTrigger'
import Tooltip from 'react-bootstrap/Tooltip'

import { dateFormat, timeFormat, timeFormatLong } from '../constants'

const DateTooltip = ({ value }: { value: string | number }) => {
  const { formatDate, formatTime } = useIntl()
  const date = new Date(time(value).from('ns').to('ms').value)
  const visibleDate = `${formatDate(
    date,
    dateFormat as Intl.DateTimeFormatOptions
  )} ${formatTime(date, timeFormat as Intl.DateTimeFormatOptions)}`
  const tooltipDate = `${formatDate(
    date,
    dateFormat as Intl.DateTimeFormatOptions
  )} ${formatTime(date, timeFormatLong as Intl.DateTimeFormatOptions)}`

  return (
    <OverlayTrigger
      placement="top"
      overlay={<Tooltip className="plgd-tooltip">{tooltipDate}</Tooltip>}
    >
      <span className="no-wrap-text tooltiped-text">{visibleDate}</span>
    </OverlayTrigger>
  )
}

DateTooltip.displayName = 'DateTooltip'

export default DateTooltip
