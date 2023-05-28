import { useIntl } from 'react-intl'
// @ts-ignore
import * as converter from 'units-converter/dist/es/index'

import { dateFormat, timeFormatLong } from '../constants'

const time = converter.time

const DateFormat = ({ value }: { value: string | number }) => {
    const { formatDate, formatTime } = useIntl()
    const date = new Date(time(value).from('ns').to('ms').value)

    return <span>{`${formatDate(date, dateFormat as Intl.DateTimeFormatOptions)} ${formatTime(date, timeFormatLong as Intl.DateTimeFormatOptions)}`}</span>
}

DateFormat.displayName = 'DateFormat'

export default DateFormat
