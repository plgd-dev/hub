import { FC, useEffect, useState, useContext } from 'react'
import ReactDOM from 'react-dom'
import { Props } from './PendingCommandsExpandableList.types'
import PendingCommandsList from '../PendingCommandsList'
import { AppContext } from '@/containers/App/AppContext'
import { motion, AnimatePresence } from 'framer-motion'

const PendingCommandsExpandableList: FC<Props> = ({ deviceId }) => {
    const [domReady, setDomReady] = useState(false)
    const { footerExpanded } = useContext(AppContext)

    useEffect(() => {
        setDomReady(true)
    }, [])

    return (
        <>
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
                                <div style={{ marginTop: 12 }}>
                                    <PendingCommandsList deviceId={deviceId} />
                                </div>
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
