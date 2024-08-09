import React, { FC, useCallback, useMemo, useState } from 'react'
import { useIntl } from 'react-intl'
import isFunction from 'lodash/isFunction'
import truncate from 'lodash/truncate'

import Modal from '@shared-ui/components/Atomic/Modal'
import FormLabel from '@shared-ui/components/Atomic/FormLabel'
import FormGroup from '@shared-ui/components/Atomic/FormGroup'
import FormInput from '@shared-ui/components/Atomic/FormInput'
import FormSelect from '@shared-ui/components/Atomic/FormSelect'
import { OptionType } from '@shared-ui/components/Atomic/FormSelect/FormSelect.types'
import Show from '@shared-ui/components/Atomic/Show'
import DatePicker from '@shared-ui/components/Atomic/DatePicker'
import { security } from '@shared-ui/common/services'
import { copyToClipboard, getApiErrorMessage } from '@shared-ui/common/utils'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import { IconCopy } from '@shared-ui/components/Atomic/Icon'
import Tooltip from '@shared-ui/components/Atomic/Tooltip'
import Alert from '@shared-ui/components/Atomic/Alert'
import Spacer from '@shared-ui/components/Atomic/Spacer'

import { messages as t } from '../ApiTokens.i18n'
import { messages as g } from '@/containers/Global.i18n'
import { Props } from './AddNewTokenModal.types'
import { dateFormat } from '@/containers/PendingCommands/constants'
import { createApiTokenApi } from '@/containers/ApiTokens/rest'
import notificationId from '@/notificationId'
import { CLIENT_ASSERTION_TYPE, GRANT_TYPE } from '@/containers/ApiTokens/constants'
import * as styles from './AddNewTokenModal.styles'

const AddNewTokenModal: FC<Props> = (props) => {
    const { dataTestId, defaultName, handleClose, onSubmit, refresh, show, showToken } = props

    const { formatMessage: _, formatDate } = useIntl()

    const expirationOptions = useMemo(
        () => [
            {
                value: 7,
                label: `7 ${_(g.days)}`,
            },
            {
                value: 30,
                label: `30 ${_(g.days)}`,
            },
            {
                value: 60,
                label: `60 ${_(g.days)}`,
            },
            {
                value: 90,
                label: `90 ${_(g.days)}`,
            },
            {
                value: -1,
                label: `${_(g.custom)}...`,
            },
            {
                value: 0,
                label: _(t.noExpiration),
            },
        ],
        [_]
    )

    const [name, setName] = useState(defaultName || '')
    const [expiration, setExpiration] = useState(30)
    const [pickedExpiration, setPickedExpiration] = useState<null | Date>(null)
    const [tokenData, setTokenData] = useState<undefined | string>(undefined)

    const minDate = useMemo(() => {
        const min = new Date()
        min.setDate(min.getDate() + 1)

        return min
    }, [])

    const currentDate = useMemo(() => {
        const cd = new Date()
        cd.setDate(cd.getDate() + expiration)

        return cd
    }, [expiration])

    const expirationVal = useMemo(() => {
        if (expiration === -1 && pickedExpiration) {
            return Math.floor(pickedExpiration.getTime() / 1000)
        }

        return Math.floor(currentDate.getTime() / 1000)
    }, [currentDate, expiration, pickedExpiration])

    const closeModal = useCallback(() => {
        handleClose()
        setTokenData(undefined)
    }, [handleClose])

    const renderBody = () => (
        <div>
            <FormGroup compactFormComponentsView={false} id='note'>
                <FormLabel required text={_(g.name)} />
                <FormInput dataTestId={dataTestId?.concat('-form-name')} onChange={(e) => setName(e.target.value)} value={name} />
            </FormGroup>
            <FormGroup compactFormComponentsView={false} id='expiration'>
                <FormLabel required text={_(t.expiration)} />
                <FormSelect
                    menuPortalTarget={document.getElementById('modal-root')}
                    menuZIndex={100}
                    onChange={(value: OptionType) => {
                        setExpiration(value.value as number)

                        if (value.value === -1) {
                            setPickedExpiration(minDate)
                        }
                    }}
                    options={expirationOptions}
                    value={expirationOptions.find((option) => option.value === expiration)}
                />
            </FormGroup>
            <Show>
                <Show.When isTrue={expiration > 0 && !tokenData}>
                    <p css={styles.expNote}>{_(t.tokenExpireOn, { date: formatDate(currentDate, dateFormat as Intl.DateTimeFormatOptions) })}</p>
                </Show.When>
                <Show.When isTrue={expiration === -1}>
                    <FormGroup compactFormComponentsView={false} id='expirationDate'>
                        <FormLabel text={_(t.expirationDate)} />
                        <DatePicker
                            bottomButtons
                            defaultValue={pickedExpiration}
                            i18n={{
                                clear: _(g.clear),
                                confirm: _(g.confirm),
                            }}
                            minDate={minDate}
                            onChange={(d) => setPickedExpiration(d)}
                        />
                    </FormGroup>
                </Show.When>
            </Show>
            <Show>
                <Show.When isTrue={!!showToken && !!tokenData}>
                    <Alert dataTestId={dataTestId?.concat('-alert')}>{_(t.tokenExpNote)}</Alert>
                    <Spacer type='mt-4'>
                        <div css={styles.copyBox} data-test-id={dataTestId?.concat('-copy')}>
                            {truncate(tokenData, { length: 60 })}
                            <Tooltip content={_(g.copy)} css={styles.copyIcon} id='copyIcon' portalTarget={undefined}>
                                <IconCopy onClick={() => copyToClipboard(tokenData || '')} />
                            </Tooltip>
                        </div>
                    </Spacer>
                </Show.When>
            </Show>
        </div>
    )

    const resetForm = useCallback(() => {
        setName(defaultName || '')
        setExpiration(30)
    }, [defaultName])

    const handleSubmit = async () => {
        try {
            const { m2mOauthClient } = security.getWellKnownConfig()

            const { data } = await createApiTokenApi({
                clientId: m2mOauthClient.clientId,
                expiration: expirationVal,
                clientAssertionType: CLIENT_ASSERTION_TYPE,
                clientAssertion: security.getAccessToken(),
                tokenName: name,
                grantType: GRANT_TYPE,
            })

            setTokenData(data.accessToken)

            isFunction(refresh) && refresh()
            isFunction(onSubmit) && onSubmit(data, expirationVal)

            Notification.success(
                {
                    title: _(t.addApiTokenSuccess),
                    message: _(t.addApiTokenSuccessMessage, { name }),
                },
                { notificationId: notificationId.HUB_API_TOKENS_LIST_ADD_SUCCESS }
            )

            if (!showToken) {
                handleClose()
            }
            resetForm()
        } catch (error: any) {
            Notification.error(
                { title: _(t.addTokenError), message: getApiErrorMessage(error) },
                { notificationId: notificationId.HUB_API_TOKENS_LIST_ADD_ERROR }
            )
        }
    }

    return (
        <Modal
            appRoot={document.getElementById('root')}
            closeButtonText={_(g.close)}
            dataTestId={dataTestId}
            footerActions={[
                {
                    dataTestId: dataTestId?.concat('-reset'),
                    label: _(g.close),
                    onClick: () => {
                        closeModal()
                        resetForm()
                    },
                    variant: 'secondary',
                },
                {
                    dataTestId: dataTestId?.concat('-generate'),
                    label: _(t.generateToken),
                    disabled: !name,
                    onClick: () => {
                        handleSubmit().then(() => {})
                        resetForm()
                    },
                    variant: 'primary',
                },
            ]}
            onClose={() => {
                closeModal()
                resetForm()
            }}
            portalTarget={document.getElementById('modal-root')}
            renderBody={renderBody}
            show={show}
            title={_(t.generateNewToken)}
            zIndex={99}
        />
    )
}

AddNewTokenModal.displayName = 'AddNewTokenModal'

export default AddNewTokenModal
