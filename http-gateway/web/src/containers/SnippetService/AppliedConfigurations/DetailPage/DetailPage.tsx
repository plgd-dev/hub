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
import Loadable from '@shared-ui/components/Atomic/Loadable'
import Headline from '@shared-ui/components/Atomic/Headline'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import SimpleStripTable from '@shared-ui/components/Atomic/SimpleStripTable'
import ResourceToggleCreator from '@shared-ui/components/Organisms/ResourceToggleCreator'
import { ResourceType } from '@shared-ui/components/Organisms/ResourceToggleCreator/ResourceToggleCreator.types'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { pages } from '@/routes'
import { useAppliedConfigurationDetail } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { getResourceI18n } from '@/containers/SnippetService/utils'

const DetailPage: FC<any> = () => {
    const { appliedConfigurationId } = useParams()

    const { formatMessage: _ } = useIntl()
    const { data, loading, error } = useAppliedConfigurationDetail(appliedConfigurationId || '', !!appliedConfigurationId)

    const [activeItem, setActiveItem] = useState('0')
    const [pageLoading, setPageLoading] = useState(false)

    const { refsByKey, setRef } = useRefs()

    const breadcrumbs = useMemo(
        () => [
            { label: _(confT.snippetService), link: generatePath(pages.SNIPPET_SERVICE.LINK) },
            { label: _(confT.appliedConfiguration), link: generatePath(pages.SNIPPET_SERVICE.APPLIED_CONFIGURATIONS.LINK) },
            { label: data?.name || '' },
        ],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [data]
    )

    useEffect(() => {
        if (error) {
            Notification.error(
                { title: _(confT.conditionsError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_SNIPPET_SERVICE_APPLIED_CONFIGURATIONS_DETAIL_PAGE_ERROR }
            )
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [error])

    const menu = useMemo(
        () => [
            { id: '0', link: '#general', title: _(g.general), icon: <IconInfo /> },
            { id: '1', link: '#listOfResources', title: _(confT.listOfResources), icon: <IconShield /> },
            { id: '2', link: '#appliedToDevices', title: _(confT.appliedToDevices), icon: <IconGlobe /> },
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

            setTimeout(() => {
                element?.scrollIntoView({ behavior: 'smooth' })
            }, 0)
        },
        [refs]
    )

    const resourceI18n = useMemo(() => getResourceI18n(_), [_])

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
                    </Column>
                    {isDesktopOrLaptop && <Column xl={1}></Column>}
                    <Column style={isDesktopOrLaptop ? { height: '100%' } : { flex: '1 1 auto', overflow: 'hidden' }} xl={8}>
                        <Spacer style={{ height: '100%', overflow: 'auto' }} type='pr-10'>
                            <Loadable condition={!loading && !!data}>
                                <>
                                    <div ref={(element: HTMLDivElement) => setRef(element, '0')}>
                                        <Spacer type='mb-4'>
                                            <Headline type='h5'>{_(g.general)}</Headline>
                                        </Spacer>

                                        <SimpleStripTable
                                            leftColSize={6}
                                            rightColSize={6}
                                            rows={[
                                                {
                                                    attribute: _(g.name),
                                                    value: (
                                                        <FormGroup id='name' marginBottom={false} style={{ width: '100%' }}>
                                                            <FormInput disabled align='right' size='small' value={data?.name} />
                                                        </FormGroup>
                                                    ),
                                                },
                                            ]}
                                        />
                                    </div>

                                    <div ref={(element: HTMLDivElement) => setRef(element, '1')}>
                                        <Spacer type='mt-8'>
                                            <Headline type='h5'>{_(g.listOfResources)}</Headline>
                                            <p style={{ margin: '4px 0 0 0' }}>Short description...</p>
                                        </Spacer>

                                        <Spacer type='mt-6'>
                                            {data?.resources &&
                                                data?.resources?.map((resource: ResourceType, key: number) => (
                                                    <Spacer key={key} type='mb-2'>
                                                        <ResourceToggleCreator defaultOpen readOnly i18n={resourceI18n} resourceData={resource} />
                                                    </Spacer>
                                                ))}
                                        </Spacer>
                                    </div>

                                    <div ref={(element: HTMLDivElement) => setRef(element, '2')}>
                                        <Spacer type='mt-8'>
                                            <Headline type='h5'>{_(confT.appliedToDevices)}</Headline>
                                            <p style={{ margin: '4px 0 0 0' }}>Short description...</p>
                                        </Spacer>
                                    </div>
                                </>
                            </Loadable>
                        </Spacer>
                    </Column>
                </Row>
            </Spacer>
        </PageLayout>
    )
}

DetailPage.displayName = 'DetailPage'

export default DetailPage
