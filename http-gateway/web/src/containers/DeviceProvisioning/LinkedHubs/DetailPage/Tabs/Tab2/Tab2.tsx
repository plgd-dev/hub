import React, { FC, lazy, useCallback, useMemo, useRef, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'

import Column from '@shared-ui/components/Atomic/Grid/Column'
import Row from '@shared-ui/components/Atomic/Grid/Row'
import ContentMenu from '@shared-ui/components/Atomic/ContentMenu'
import IconInfo from '@shared-ui/components/Atomic/Icon/components/IconInfo'
import ContentSwitch from '@shared-ui/components/Atomic/ContentSwitch'
import { ItemType } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'
import IconShield from '@shared-ui/components/Atomic/Icon/components/IconShield'

import { messages as g } from '../../../../../Global.i18n'
import { messages as t } from '../../../LinkedHubs.i18n'
import { Props } from './Tab2.types'
import { pages } from '@/routes'

const TabContent1 = lazy(() => import('./Contents/TabContent1'))
const TabContent2 = lazy(() => import('./Contents/TabContent2'))

const Tab2: FC<Props> = (props) => {
    const { defaultFormData, loading } = props
    const { formatMessage: _ } = useIntl()
    const { section, hubId } = useParams()

    const navigate = useNavigate()

    const contentRef1 = useRef<HTMLHeadingElement | null>(null)
    const contentRef2 = useRef<HTMLHeadingElement | null>(null)
    const contentRef3 = useRef<HTMLHeadingElement | null>(null)

    const menu = useMemo(
        () => [
            { id: '0', icon: <IconInfo />, link: '', title: _(t.generalKeepAlive) },
            {
                id: '1',
                icon: <IconShield />,
                link: 'tls',
                title: _(g.tls),
                children: [
                    { id: '10', title: _(t.caPool), contentRef: contentRef1 },
                    { id: '11', title: _(t.privateKey), contentRef: contentRef2 },
                    { id: '12', title: _(t.certificate), contentRef: contentRef3 },
                ],
            },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        []
    )

    const [activeItem, setActiveItem] = useState(menu.find((item) => item.link === `${section}`)?.id || '0')

    const handleItemClick = useCallback(
        (item: ItemType) => {
            setActiveItem(item.id)

            navigate(generatePath(pages.DPS.LINKED_HUBS.DETAIL.LINK, { hubId: hubId, tab: pages.DPS.LINKED_HUBS.DETAIL.TABS[1], section: item.link }))
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
                    <TabContent2
                        contentRefs={{
                            ref1: contentRef1,
                            ref2: contentRef2,
                            ref3: contentRef3,
                        }}
                        defaultFormData={defaultFormData}
                        loading={loading}
                    />
                </ContentSwitch>
            </Column>
        </Row>
    )
}

Tab2.displayName = 'Tab2'

export default Tab2
