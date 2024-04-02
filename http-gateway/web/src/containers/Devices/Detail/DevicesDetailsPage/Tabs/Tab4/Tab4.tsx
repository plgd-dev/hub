import { FC, lazy, useCallback, useMemo, useState } from 'react'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import { useIntl } from 'react-intl'

import Row from '@shared-ui/components/Atomic/Grid/Row'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import ContentMenu from '@shared-ui/components/Atomic/ContentMenu'
import { ItemType } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import Loadable from '@shared-ui/components/Atomic/Loadable'

import { Props } from './Tab4.types'
import { pages } from '@/routes'
import { messages as g } from '@/containers/Global.i18n'
import { messages as t } from '../../../../Devices.i18n'
import { getStatusFromCode } from '@/containers/DeviceProvisioning/utils'
import isEmpty from 'lodash/isEmpty'

const TabContent1 = lazy(() => import('../../../../../DeviceProvisioning/ProvisioningRecords/DetailPage/Tabs/Tab1'))
const TabContent2 = lazy(() => import('../../../../../DeviceProvisioning/ProvisioningRecords/DetailPage/Tabs/Tab2'))
const TabContent3 = lazy(() => import('../../../../../DeviceProvisioning/ProvisioningRecords/DetailPage/Tabs/Tab3'))

const Tab4: FC<Props> = (props) => {
    const { provisioningRecords } = props

    const { formatMessage: _ } = useIntl()
    const { section, id } = useParams()
    const navigate = useNavigate()

    const menu = useMemo(
        () => [
            { id: '0', link: pages.DEVICES.DETAIL.SECTIONS[0], title: _(g.details) },
            {
                id: '1',
                link: pages.DEVICES.DETAIL.SECTIONS[1],
                status: provisioningRecords && provisioningRecords.credential ? getStatusFromCode(provisioningRecords.credential.status.coapCode) : undefined,
                title: _(t.credentials),
            },
            {
                id: '2',
                link: pages.DEVICES.DETAIL.SECTIONS[2],
                status: provisioningRecords && provisioningRecords.acl ? getStatusFromCode(provisioningRecords.acl.status.coapCode) : undefined,
                title: _(t.acls),
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [provisioningRecords]
    )

    const [activeItem, setActiveItem] = useState(menu.find((item) => item.link === `${section}`)?.id || '0')

    const handleItemClick = useCallback(
        (item: ItemType) => {
            setActiveItem(item.id)
            navigate(generatePath(pages.DEVICES.DETAIL.LINK, { id, tab: pages.DEVICES.DETAIL.TABS[3], section: item.link }))
        },
        [id, navigate]
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
                <Loadable condition={provisioningRecords}>
                    <ContentSwitch activeItem={parseInt(activeItem)}>
                        <TabContent1 isDeviceMode data={provisioningRecords} />
                        <TabContent2 data={provisioningRecords} />
                        <TabContent3 data={provisioningRecords} />
                    </ContentSwitch>
                </Loadable>
            </Column>
        </Row>
    )
}

Tab4.displayName = 'Tab4'

export default Tab4
