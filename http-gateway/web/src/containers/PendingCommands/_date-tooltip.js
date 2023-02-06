import PropTypes from 'prop-types'
import { useIntl } from 'react-intl'
import { time } from 'units-converter'
import OverlayTrigger from 'react-bootstrap/OverlayTrigger'
import Tooltip from 'react-bootstrap/Tooltip'

import { dateFormat, timeFormat, timeFormatLong } from './constants'

export const DateTooltip = ({ value }) => {
  const { formatDate, formatTime } = useIntl()
  const date = new Date(time(value).from('ns').to('ms').value)
  const visibleDate = `${formatDate(date, dateFormat)} ${formatTime(
    date,
    timeFormat
  )}`
  const tooltipDate = `${formatDate(date, dateFormat)} ${formatTime(
    date,
    timeFormatLong
  )}`

  return (
    <OverlayTrigger
      placement="top"
      overlay={<Tooltip className="plgd-tooltip">{tooltipDate}</Tooltip>}
    >
      <span className="no-wrap-text tooltiped-text">{visibleDate}</span>
    </OverlayTrigger>
  )
}

DateTooltip.propTypes = {
  value: PropTypes.oneOfType([PropTypes.string, PropTypes.number]).isRequired,
}
