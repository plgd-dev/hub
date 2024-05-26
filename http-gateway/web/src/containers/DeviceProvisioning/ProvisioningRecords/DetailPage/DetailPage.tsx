import React, { FC, lazy, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'

import Row from '@shared-ui/components/Atomic/Grid/Row'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import ContentMenu from '@shared-ui/components/Atomic/ContentMenu'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import { ItemType } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'
import Headline from '@shared-ui/components/Atomic/Headline'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import { useRefs } from '@shared-ui/common/hooks/useRefs'

import { pages } from '@/routes'
import { messages as g } from '@/containers/Global.i18n'
import { getStatusFromCode } from '@/containers/DeviceProvisioning/utils'
import { messages as t } from '../ProvisioningRecords.i18n'
import { Props } from './DetailPage.types'

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
            { id: '1', link: pages.DEVICES.DETAIL.SECTIONS[1], title: _(t.cloud), status: getStatusFromCode(provisioningRecord.cloud.status.coapCode) },
            {
                id: '2',
                link: pages.DEVICES.DETAIL.SECTIONS[2],
                title: _(t.ownership),
                status: getStatusFromCode(provisioningRecord.ownership.status.coapCode),
            },
            { id: '3', link: pages.DEVICES.DETAIL.SECTIONS[3], title: _(t.timeSynchronisation) },
            {
                id: '4',
                link: pages.DEVICES.DETAIL.SECTIONS[1],
                status: provisioningRecord && provisioningRecord.credential ? getStatusFromCode(provisioningRecord.credential.status.coapCode) : undefined,
                title: _(t.credentials),
            },
            {
                id: '5',
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
        <Spacer style={{ height: '100%', overflow: 'hidden' }} type='pl-10'>
            <Row style={{ height: '100%', overflow: 'hidden' }}>
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
                <Column style={{ height: '100%' }} xl={9}>
                    <Loadable condition={!!provisioningRecord}>
                        <Spacer style={{ height: '100%', overflow: 'auto' }} type='pr-10'>
                            <div ref={(element) => setRef(element, '0')}>
                                <Tab1
                                    isDeviceMode
                                    data={provisioningRecord}
                                    refs={{
                                        cloud: (element: HTMLElement) => setRef(element, '1'),
                                        ownership: (element: HTMLElement) => setRef(element, '2'),
                                        time: (element: HTMLElement) => setRef(element, '3'),
                                    }}
                                />
                            </div>
                            <div ref={(element) => setRef(element, '4')}>
                                <Spacer type='mt-8 mb-3'>
                                    <Headline type='h5'>{_(t.credentials)}</Headline>
                                </Spacer>
                                <Tab2 data={provisioningRecord} />
                            </div>
                            <div ref={(element) => setRef(element, '5')}>
                                <Spacer type='mt-8 mb-3'>
                                    <Headline type='h5'>{_(t.acls)}</Headline>
                                </Spacer>
                                <Tab3 data={provisioningRecord} />
                            </div>
                        </Spacer>
                    </Loadable>
                </Column>
            </Row>
        </Spacer>
    )
}

DetailPage.displayName = 'DetailPage'

export default DetailPage
