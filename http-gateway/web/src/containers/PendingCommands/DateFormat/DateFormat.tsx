import { FC } from 'react'
import { useIntl } from 'react-intl'
import { Props } from './DateFormat.types'
// @ts-ignore
import * as converter from 'units-converter/dist/es/index'

import { dateFormat, timeFormatLong } from '../constants'

const time = converter.time

const DateFormat: FC<Props> = (props) => {
    const { prefixTest, rawValue, value } = props
    const { formatDate, formatTime } = useIntl()
    const date = new Date(rawValue ? value : time(value).from('ns').to('ms').value)

    return (
        <span>
            {prefixTest}
            {`${formatDate(date, dateFormat as Intl.DateTimeFormatOptions)}
            ${formatTime(date, timeFormatLong as Intl.DateTimeFormatOptions)}`}
        </span>
    )
}

DateFormat.displayName = 'DateFormat'

export default DateFormat
