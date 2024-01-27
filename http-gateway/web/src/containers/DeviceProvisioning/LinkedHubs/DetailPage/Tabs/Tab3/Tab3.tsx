import React, { FC, lazy, useCallback, useMemo, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { useNavigate, useParams } from 'react-router-dom'

import Row from '@shared-ui/components/Atomic/Grid/Row'
import Column from '@shared-ui/components/Atomic/Grid/Column'
import ContentMenu from '@shared-ui/components/Atomic/ContentMenu'
import IconInfo from '@shared-ui/components/Atomic/Icon/components/IconInfo'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import { ItemType } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'
import IconLock from '@shared-ui/components/Atomic/Icon/components/IconLock'
import IconShield from '@shared-ui/components/Atomic/Icon/components/IconShield'
import IconGlobe from '@shared-ui/components/Atomic/Icon/components/IconGlobe'

import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { getTabRoute } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import { Props } from './Tab3.types'

const TabContent1 = lazy(() => import('./Contents/TabContent1'))
const TabContent2 = lazy(() => import('./Contents/TabContent2'))
const TabContent3 = lazy(() => import('./Contents/TabContent3'))
const TabContent4 = lazy(() => import('./Contents/TabContent4'))

const Tab3: FC<Props> = (props) => {
    const { defaultFormData, loading } = props
    const { formatMessage: _ } = useIntl()
    const { section, hubId } = useParams()
    const navigate = useNavigate()

    const contentRef1 = useRef<HTMLHeadingElement | null>(null)
    const contentRef2 = useRef<HTMLHeadingElement | null>(null)
    const contentRef3 = useRef<HTMLHeadingElement | null>(null)
    const contentRef4 = useRef<HTMLHeadingElement | null>(null)

    const menu = useMemo(
        () => [
            { id: '0', link: '', icon: <IconInfo />, title: _(t.general) },
            { id: '1', link: '/oauth-client', icon: <IconLock />, title: _(t.oAuthClient) },
            {
                id: '2',
                link: '/tls',
                title: _(g.tls),
                icon: <IconShield />,
                children: [
                    { id: '20', title: _(t.caPool), contentRef: contentRef1 },
                    { id: '21', title: _(t.privateKey), contentRef: contentRef2 },
                    { id: '22', title: _(t.certificate), contentRef: contentRef3 },
                    { id: '23', title: _(t.useSystemCAPool), contentRef: contentRef4 },
                ],
            },
            { id: '3', link: '/http', icon: <IconGlobe />, title: _(t.hTTP) },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const [activeItem, setActiveItem] = useState(menu.find((item) => item.link === `/${section}`)?.id || '0')

    const handleItemClick = useCallback(
        (item: ItemType) => {
            setActiveItem(item.id)
            console.log(item)

            navigate(`/device-provisioning/linked-hubs/${hubId}${getTabRoute(2)}${item.link}`, { replace: true })
        },
        [hubId, navigate]
    )

    return (
        <Row
            style={{
                height: '100%',
            }}
        >
            <Column size={3}>
                <ContentMenu
                    activeItem={activeItem}
                    handleItemClick={handleItemClick}
                    handleSubItemClick={(_item, parentItem) => {
                        _item.contentRef?.current?.scrollIntoView({ behavior: 'smooth' })
                        setActiveItem(parentItem.id)
                    }}
                    menu={menu}
                    title={_(g.navigation)}
                />
            </Column>
            <Column size={1}></Column>
            <Column
                size={8}
                style={{
                    height: '100%',
                    overflow: 'auto',
                }}
            >
                <ContentSwitch activeItem={parseInt(activeItem)}>
                    <TabContent1 defaultFormData={defaultFormData} loading={loading} />
                    <TabContent2 defaultFormData={defaultFormData} loading={loading} />
                    <TabContent3
                        contentRefs={{
                            ref1: contentRef1,
                            ref2: contentRef2,
                            ref3: contentRef3,
                            ref4: contentRef4,
                        }}
                        defaultFormData={defaultFormData}
                        loading={loading}
                    />
                    <TabContent4 defaultFormData={defaultFormData} loading={loading} />
                </ContentSwitch>
            </Column>
        </Row>
    )
}

Tab3.displayName = 'Tab3'

export default Tab3
