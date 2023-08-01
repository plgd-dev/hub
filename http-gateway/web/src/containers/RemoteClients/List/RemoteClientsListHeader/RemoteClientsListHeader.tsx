import { FC } from 'react'
import { useIntl } from 'react-intl'

import Button from '@shared-ui/components/Atomic/Button'
import IconPlus from '@shared-ui/components/Atomic/Icon/components/IconPlus'

import * as styles from '@/containers/Devices/List/DevicesListHeader/DevicesListHeader.styles'
import { messages as t } from '../../RemoteClients.i18n'
import { Props } from './RemoteClientsListHeader.types'

const RemoteClientsListHeader: FC<Props> = (props) => {
    const { dataLoading, onClientClick } = props
    const { formatMessage: _ } = useIntl()
    return (
        <Button css={styles.item} disabled={dataLoading} icon={<IconPlus />} onClick={onClientClick} variant='primary'>
            {_(t.client)}
        </Button>
    )
}

RemoteClientsListHeader.displayName = 'RemoteClientsListHeader'

export default RemoteClientsListHeader
