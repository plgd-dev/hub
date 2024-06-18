import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useParams } from 'react-router-dom'

import PageLayout from '@/containers/Common/PageLayout'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { useRefs } from '@shared-ui/common/hooks/useRefs'
import IconInfo from '@shared-ui/components/Atomic/Icon/components/IconInfo'
import IconShield from '@shared-ui/components/Atomic/Icon/components/IconShield'
import IconGlobe from '@shared-ui/components/Atomic/Icon/components/IconGlobe'
import { ItemType, SubItemItem } from '@shared-ui/components/Atomic/ContentMenu/ContentMenu.types'
import Spacer from '@shared-ui/components/Atomic/Spacer'
import Row from '@shared-ui/components/Atomic/Grid/Row'
import ContentMenu from '@shared-ui/components/Atomic/ContentMenu'
import Column from '@shared-ui/components/Atomic/Grid/Column'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { pages } from '@/routes'
import { useAppliedDeviceConfigDetail } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Headline from '@shared-ui/components/Atomic/Headline'

const DetailPage: FC<any> = () => {
    const { appliedDeviceConfigId } = useParams()

    const { formatMessage: _ } = useIntl()
    const { data, loading, error } = useAppliedDeviceConfigDetail(appliedDeviceConfigId!, !!appliedDeviceConfigId)

    const [activeItem, setActiveItem] = useState('0')
    const [pageLoading, setPageLoading] = useState(false)

    const { refsByKey, setRef } = useRefs()

    const breadcrumbs = useMemo(
        () => [
            { label: _(confT.snippetService), link: generatePath(pages.SNIPPET_SERVICE.LINK) },
            { label: _(confT.conditions), link: generatePath(pages.SNIPPET_SERVICE.APPLIED_DEVICE_CONFIG.LINK) },
            { label: data?.name || '' },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data]
    )

    useEffect(() => {
        if (error) {
            Notification.error(
                { title: _(confT.conditionsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_DEVICE_DETAIL_PAGE_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const menu = useMemo(
        () => [
            { id: '0', link: '#general', title: _(g.general), icon: <IconInfo /> },
            { id: '1', link: '#filters', title: _(g.filters), icon: <IconShield /> },
            { id: '2', link: '#APIAccessToken', title: _(confT.APIAccessToken), icon: <IconGlobe /> },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data]
    )

    const refs = Object.values(refsByKey).filter(Boolean)

    const handleItemClick = useCallback(
        (item: ItemType | SubItemItem) => {
            setActiveItem(item.id)
            const id = parseInt(item.id)
            const element = refs[id] as HTMLElement

            element?.scrollIntoView({ behavior: 'smooth' })
        },
        [refs]
    )

    const isDesktopOrLaptop = true

    return (
        <PageLayout breadcrumbs={breadcrumbs} loading={loading || pageLoading} title={data?.name} xPadding={false}>
            <Spacer style={{ height: '100%', overflow: 'hidden' }} type='pt-4 pl-10'>
                <Row style={{ height: '100%' }}>
                    <Column xl={3}>
                        <Spacer type={`mb-4${isDesktopOrLaptop ? '' : ' pr-10'}`}>
                            <ContentMenu
                                activeItem={activeItem}
                                handleItemClick={handleItemClick}
                                handleSubItemClick={(subItem, parentItem) => handleItemClick(subItem)}
                                menu={menu}
                                title={_(g.navigation)}
                            />
                        </Spacer>
                        {isDesktopOrLaptop && <Column xl={1}></Column>}
                        <Column style={isDesktopOrLaptop ? { height: '100%' } : { flex: '1 1 auto', overflow: 'hidden' }} xl={8}>
                            <Spacer style={{ height: '100%', overflow: 'auto' }} type='pr-10'>
                                <Loadable condition={!loading && !!data}>
                                    <Spacer type='mb-4'>
                                        <Headline type='h5'>{_(g.general)}</Headline>
                                    </Spacer>
                                </Loadable>
                            </Spacer>
                        </Column>
                    </Column>
                </Row>
            </Spacer>
        </PageLayout>
    )
}

DetailPage.displayName = 'DetailPage'

export default DetailPage
