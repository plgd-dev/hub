import { FC, useEffect, useState, useContext } from 'react'
import ReactDOM from 'react-dom'
import { useIntl } from 'react-intl'
import { Props } from './PendingCommandsExpandableList.types'
import PendingCommandsList from '../PendingCommandsList'
import { messages as t } from '../PendingCommands.i18n'
import { AppContext } from '@/containers/App/AppContext'
import { motion, AnimatePresence } from 'framer-motion'
import isFunction from 'lodash/isFunction'

const PendingCommandsExpandableList: FC<Props> = ({ deviceId }) => {
    const [domReady, setDomReady] = useState(false)
    const [test, setTest] = useState(0)
    const { formatMessage: _ } = useIntl()
    const { footerExpanded, setFooterExpanded, collapsed } = useContext(AppContext)

    useEffect(() => {
        setDomReady(true)
        setTest(Math.random)
    }, [footerExpanded, collapsed])

    return (
        <>
            {domReady &&
                ReactDOM.createPortal(
                    <span
                        data-a={test}
                        onClick={() => {
                            isFunction(setFooterExpanded) && setFooterExpanded(!footerExpanded)
                        }}
                    >
                        {_(t.recentTasks)}
                    </span>,
                    document.querySelector('#recentTasksPortalTitleTarget') as Element
                )}
            {domReady &&
                footerExpanded &&
                ReactDOM.createPortal(
                    <AnimatePresence mode='wait'>
                        {footerExpanded && (
                            <motion.div
                                layout
                                animate={{ opacity: 1 }}
                                exit={{
                                    opacity: 0,
                                }}
                                initial={{ opacity: 0 }}
                                transition={{
                                    duration: 0.3,
                                }}
                            >
                                <PendingCommandsList deviceId={deviceId} />
                            </motion.div>
                        )}
                    </AnimatePresence>,
                    document.querySelector('#recentTasksPortalTarget') as Element
                )}
        </>
    )
}

PendingCommandsExpandableList.displayName = 'PendingCommandsExpandableList'

export default PendingCommandsExpandableList
