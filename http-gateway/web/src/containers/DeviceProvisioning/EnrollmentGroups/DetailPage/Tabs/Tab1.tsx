import { FC } from 'react'
import { useIntl } from 'react-intl'

import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'

import { Props } from './Tab1.types'
import { messages as g } from '../../../../Global.i18n'

const Tab1: FC<Props> = (props) => {
    const { formatMessage: _ } = useIntl()
    const { data } = props

    console.log(data)

    return (
        <div>
            <SimpleStripTable rows={[{ attribute: _(g.name), value: data?.name }]} />
            <SimpleStripTable rows={[{ attribute: _(g.ownerID), value: data?.owner }]} />
        </div>
    )
}

Tab1.displayName = 'Tab1'

export default Tab1
