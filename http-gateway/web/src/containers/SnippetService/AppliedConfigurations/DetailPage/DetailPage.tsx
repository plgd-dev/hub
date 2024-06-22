import React, { FC, useCallback, useEffect, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import { generatePath, useNavigate, useParams } from 'react-router-dom'
import { useMediaQuery } from 'react-responsive'

import PageLayout from '@/containers/Common/PageLayout'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { getApiErrorMessage } from '@shared-ui/common/utils'
import { useRefs } from '@shared-ui/common/hooks/useRefs'
import IconInfo from '@shared-ui/components/Atomic/Icon/components/IconInfo'
import IconShield from '@shared-ui/components/Atomic/Icon/components/IconShield'
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
import StatusTag from '@shared-ui/components/Atomic/StatusTag'
import { tagVariants as statusTagVariants } from '@shared-ui/components/Atomic/StatusTag/constants'
import { tagVariants } from '@shared-ui/components/Atomic/Tag/constants'
import Tag from '@shared-ui/components/Atomic/Tag'
import IconLink from '@shared-ui/components/Atomic/Icon/components/IconLink'

import { messages as confT } from '@/containers/SnippetService/SnippetService.i18n'
import { pages } from '@/routes'
import { useAppliedConfigurationDetail } from '@/containers/SnippetService/hooks'
import notificationId from '@/notificationId'
import { messages as g } from '@/containers/Global.i18n'
import { getResourceI18n, getResourceStatusTag } from '@/containers/SnippetService/utils'
import DetailHeader from './DetailHeader'

const DetailPage: FC<any> = () => {
    const { appliedConfigurationId } = useParams()

    const { formatMessage: _ } = useIntl()
    const { data, loading, error } = useAppliedConfigurationDetail(appliedConfigurationId || '', !!appliedConfigurationId)

    const [activeItem, setActiveItem] = useState('0')

    const { refsByKey, setRef } = useRefs()
    const navigate = useNavigate()

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

    const isDesktopOrLaptop = useMediaQuery(
        {
            query: '(min-width: 1200px)',
        },
        undefined,
        () => {}
    )

    return (
        <PageLayout
            breadcrumbs={breadcrumbs}
            header={
                <DetailHeader
                    conditionId={data?.conditionId?.id || _(confT.onDemand)}
                    conditionName={data?.conditionName !== -1 ? data?.conditionName : undefined}
                    configurationId={data?.configurationId?.id || ''}
                    configurationName={data?.configurationName || ''}
                    id={data?.id || ''}
                    loading={loading}
                />
            }
            loading={loading}
            title={data?.name}
            xPadding={false}
        >
            <Spacer style={{ height: '100%', overflow: 'hidden' }} type='pt-4 pl-10'>
                <Row
                    style={
                        isDesktopOrLaptop
                            ? { height: '100%' }
                            : { display: 'flex', flexDirection: 'column', flexWrap: 'nowrap', overflow: 'hidden', height: '100%' }
                    }
                >
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
                                                    attribute: _(g.deviceName),
                                                    value: (
                                                        <FormGroup id='name' marginBottom={false} style={{ width: '100%' }}>
                                                            <FormInput disabled align='right' size='small' value={data?.name} />
                                                        </FormGroup>
                                                    ),
                                                },
                                                {
                                                    attribute: _(confT.configuration),
                                                    value: (
                                                        <Tag
                                                            onClick={() =>
                                                                navigate(
                                                                    generatePath(pages.SNIPPET_SERVICE.CONFIGURATIONS.DETAIL.LINK, {
                                                                        configurationId: data?.configurationId?.id,
                                                                        tab: '',
                                                                    })
                                                                )
                                                            }
                                                            variant={tagVariants.BLUE}
                                                        >
                                                            <IconLink />
                                                            <Spacer type='ml-2'>{data?.configurationName}</Spacer>
                                                        </Tag>
                                                    ),
                                                },
                                                {
                                                    attribute: _(confT.condition),
                                                    value: data?.conditionId ? (
                                                        <Tag
                                                            onClick={() =>
                                                                navigate(
                                                                    generatePath(pages.SNIPPET_SERVICE.CONDITIONS.DETAIL.LINK, {
                                                                        conditionId: data?.conditionId?.id,
                                                                        tab: '',
                                                                    })
                                                                )
                                                            }
                                                            variant={tagVariants.BLUE}
                                                        >
                                                            <IconLink />
                                                            <Spacer type='ml-2'>{data?.conditionName}</Spacer>
                                                        </Tag>
                                                    ) : (
                                                        <StatusTag variant={statusTagVariants.NORMAL}>{_(confT.onDemand)}</StatusTag>
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
                                                data?.resources?.map((resource, key: number) => (
                                                    <Spacer key={resource.href} type='mb-2'>
                                                        <ResourceToggleCreator
                                                            defaultOpen
                                                            readOnly
                                                            i18n={resourceI18n}
                                                            resourceData={resource}
                                                            statusTag={getResourceStatusTag(resource)}
                                                        />
                                                    </Spacer>
                                                ))}
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
