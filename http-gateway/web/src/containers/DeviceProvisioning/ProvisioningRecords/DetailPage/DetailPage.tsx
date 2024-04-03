import { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import isFunction from 'lodash/isFunction'

import Row from '@shared-ui/components/Atomic/Grid/Row'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import ContentMenu from '@shared-ui/components/Atomic/ContentMenu'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import { ItemType } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'

import { pages } from '@/routes'
import { messages as g } from '@/containers/Global.i18n'
import { getStatusFromCode } from '@/containers/DeviceProvisioning/utils'
import { messages as t } from '@/containers/Devices/Devices.i18n'
import { Props } from './DetailPage.types'

const Tab1 = lazy(() => import('./Tabs/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2'))
const Tab3 = lazy(() => import('./Tabs/Tab3'))

const DetailPage: FC<Props> = (props) => {
    const { currentTab, onItemClick, provisioningRecord } = props
    const { formatMessage: _ } = useIntl()

    const menu = useMemo(
        () => [
            { id: '0', link: pages.DEVICES.DETAIL.SECTIONS[0], title: _(g.details) },
            {
                id: '1',
                link: pages.DEVICES.DETAIL.SECTIONS[1],
                status: provisioningRecord && provisioningRecord.credential ? getStatusFromCode(provisioningRecord.credential.status.coapCode) : undefined,
                title: _(t.credentials),
            },
            {
                id: '2',
                link: pages.DEVICES.DETAIL.SECTIONS[2],
                status: provisioningRecord && provisioningRecord.acl ? getStatusFromCode(provisioningRecord.acl.status.coapCode) : undefined,
                title: _(t.acls),
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [provisioningRecord]
    )

    const [activeItem, setActiveItem] = useState(menu.find((item) => item.link === `${currentTab}`)?.id || '0')

    const handleItemClick = useCallback(
        (item: ItemType) => {
            setActiveItem(item.id)
            isFunction(onItemClick) && onItemClick(item)
        },
        [onItemClick]
    )

    return (
        <Row>
            <Column xl={3}>
                <ContentMenu
                    activeItem={activeItem}
                    handleItemClick={handleItemClick}
                    handleSubItemClick={(_item, parentItem) => {
                        setActiveItem(parentItem.id)
                    }}
                    menu={menu}
                />
            </Column>
            <Column xl={9}>
                <Loadable condition={!!provisioningRecord}>
                    <ContentSwitch activeItem={parseInt(activeItem)}>
                        <Tab1 isDeviceMode data={provisioningRecord} />
                        <Tab2 data={provisioningRecord} />
                        <Tab3 data={provisioningRecord} />
                    </ContentSwitch>
                </Loadable>
            </Column>
        </Row>
    )
}

DetailPage.displayName = 'DetailPage'

export default DetailPage
