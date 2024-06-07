import React, { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import Row from '@shared-ui/components/Atomic/Grid/Row'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import ContentMenu from '@shared-ui/components/Atomic/ContentMenu'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { ItemType } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import { useRefs } from '@shared-ui/common/hooks/useRefs'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'

import { pages } from '@/routes'
import { messages as g } from '@/containers/Global.i18n'
import { getStatusFromCode } from '@/containers/DeviceProvisioning/utils'
import { messages as t } from '../ProvisioningRecords.i18n'
import { Props } from './DetailPage.types'
import OverviewRow from './OverviewRow'

const Tab1 = lazy(() => import('./Tabs/Tab1'))
const Tab2 = lazy(() => import('./Tabs/Tab2'))
const Tab3 = lazy(() => import('./Tabs/Tab3'))

const DetailPage: FC<Props> = (props) => {
    const { currentTab, provisioningRecord } = props
    const { formatMessage: _ } = useIntl()
    const { refsByKey, setRef } = useRefs()

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

    const refs = Object.values(refsByKey).filter(Boolean)

    const handleItemClick = useCallback(
        (item: ItemType) => {
            setActiveItem(item.id)
            const id = parseInt(item.id)
            const element = refs[id] as HTMLElement

            element?.scrollIntoView({ behavior: 'smooth' })
        },
        [refs]
    )

    return (
        <div style={{ height: '100%', overflow: 'hidden' }}>
            <OverviewRow data={provisioningRecord} />
            <Spacer type='pt-8'>
                <Row style={{ height: '100%', overflow: 'hidden' }}>
                    <Column xl={3}>
                        <Spacer type='mb-4'>
                            <ContentMenu
                                activeItem={activeItem}
                                handleItemClick={handleItemClick}
                                handleSubItemClick={(_item, parentItem) => {
                                    setActiveItem(parentItem.id)
                                }}
                                menu={menu}
                            />
                        </Spacer>
                    </Column>
                    <Column style={{ height: '100%' }} xl={1}></Column>
                    <Column style={{ height: '100%' }} xl={8}>
                        <Loadable condition={!!provisioningRecord}>
                            <ContentSwitch activeItem={parseInt(activeItem, 10)}>
                                <Tab1
                                    isDeviceMode
                                    data={provisioningRecord}
                                    refs={{
                                        cloud: (element: HTMLElement) => setRef(element, '1'),
                                        ownership: (element: HTMLElement) => setRef(element, '2'),
                                        time: (element: HTMLElement) => setRef(element, '3'),
                                    }}
                                />
                                <Tab2 data={provisioningRecord} />
                                <Tab3 data={provisioningRecord} />
                            </ContentSwitch>
                        </Loadable>
                    </Column>
                </Row>
            </Spacer>
        </div>
    )
}

DetailPage.displayName = 'DetailPage'

export default DetailPage
