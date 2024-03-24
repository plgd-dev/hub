import React, { forwardRef, useContext, useEffect, useState } from 'react'
import { useIntl } from 'react-intl'
import isFunction from 'lodash/isFunction'
import ReactDOM from 'react-dom'
import { useRecoilState } from 'recoil'

import { default as PageLayoutShared } from '@shared-ui/components/Atomic/PageLayout/PageLayout'
import Footer from '@shared-ui/components/Layout/Footer'
import AppContext from '@shared-ui/app/share/AppContext'
import Breadcrumbs from '@shared-ui/components/Layout/Header/Breadcrumbs'
import NotFoundPage from '@shared-ui/components/Templates/NotFoundPage'

import { Props } from './PageLayout.types'
import { messages as g } from '@/containers/Global.i18n'
import { PendingCommandsExpandableList } from '@/containers/PendingCommands'
import { dirtyFormState, promptBlockState } from '@/store/recoil.store'
import { messages as t } from '@/containers/App/App.i18n'

const PageLayout = forwardRef<HTMLDivElement, Props>((props, ref) => {
    const { formatMessage: _ } = useIntl()
    const { children, breadcrumbs, deviceId, notFound, pendingCommands, innerPortalTarget, size, ...rest } = props
    const { footerExpanded, setFooterExpanded, collapsed } = useContext(AppContext)

    const [isDomReady, setIsDomReady] = useState(false)

    const [dirtyState] = useRecoilState(dirtyFormState)
    const [_block, setBlock] = useRecoilState(promptBlockState)

    useEffect(() => {
        setIsDomReady(true)
    }, [])

    return (
        <PageLayoutShared
            {...rest}
            collapsed={collapsed}
            footer={
                <Footer
                    footerExpanded={footerExpanded}
                    innerPortalTarget={innerPortalTarget}
                    paginationComponent={<div id='paginationPortalTarget'></div>}
                    recentTasksPortal={pendingCommands && <div id='recentTasksPortalTarget'></div>}
                    recentTasksPortalTitle={
                        pendingCommands && (
                            <span
                                id='recentTasksPortalTitleTarget'
                                onClick={() => {
                                    isFunction(setFooterExpanded) && setFooterExpanded(!footerExpanded)
                                }}
                            >
                                {_(g.pendingCommands)}
                            </span>
                        )
                    }
                    setFooterExpanded={setFooterExpanded}
                    size={size}
                />
            }
            header={notFound ? undefined : rest.header}
            ref={ref}
            title={notFound ? undefined : rest.title}
        >
            {isDomReady &&
                ReactDOM.createPortal(
                    <Breadcrumbs
                        items={breadcrumbs}
                        onItemClick={(item, e) => {
                            if (dirtyState) {
                                e.preventDefault()
                                e.stopPropagation()
                                setBlock({ link: item.link || '' })
                            }
                        }}
                    />,
                    document.querySelector('#breadcrumbsPortalTarget') as Element
                )}
            {pendingCommands && <PendingCommandsExpandableList deviceId={deviceId} />}
            {notFound ? <NotFoundPage message={_(t.notFoundPageDefaultMessage)} title={_(t.pageTitle)} /> : children}
        </PageLayoutShared>
    )
})

PageLayout.displayName = 'PageLayout'

export default PageLayout
