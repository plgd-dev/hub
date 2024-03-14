import React, { FC, useCallback, useState } from 'react'
import { useIntl } from 'react-intl'
import { Controller } from 'react-hook-form'
import cloneDeep from 'lodash/cloneDeep'

import Spacer from '@shared-ui/components/Atomic/Spacer'
import TileToggle from '@shared-ui/components/Atomic/TileToggle'
import Dropzone from '@shared-ui/components/Atomic/Dropzone'
import CaPoolModal from '@shared-ui/components/Organisms/CaPoolModal'
import CaList from '@shared-ui/components/Organisms/CaList/CaList'
import { useCaData } from '@shared-ui/common/hooks/useCaData'
import Notification from '@shared-ui/components/Atomic/Notification/Toast'
import Loadable from '@shared-ui/components/Atomic/Loadable'

import * as commonStyles from '@/containers/DeviceProvisioning/LinkedHubs/LinkNewHubPage/LinkNewHubPage.styles'
import { messages as t } from '@/containers/DeviceProvisioning/LinkedHubs/LinkedHubs.i18n'
import { pemToString, useCaI18n } from '@/containers/DeviceProvisioning/LinkedHubs/utils'
import { messages as g } from '@/containers/Global.i18n'
import notificationId from '@/notificationId'
import { Props } from './SubStepTls.types'

const SubStepTls: FC<Props> = (props) => {
    const { control, setValue, updateField, watch, prefix } = props

    const { formatMessage: _ } = useIntl()
    const i18n = useCaI18n()

    const caPool = watch(`${prefix}tls.caPool`)
    const key = watch(`${prefix}tls.key`)
    const cert = watch(`${prefix}tls.cert`)

    function nameLengthValidator(file: any, privateKey = false) {
        const format = file.name.split('.').pop()

        if ((privateKey && !['pem', 'key'].includes(format)) || (!privateKey && !['pem', 'crt', 'cer'].includes(format))) {
            return {
                code: 'invalid-format',
                message: `Bad file format`,
            }
        }
        return null
    }

    const { parsedData: caPoolData } = useCaData({
        data: caPool,
        onError: (e) =>
            Notification.error(
                { title: _(t.certificationParsingError), message: e },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_ADD_PAGE_CERT_PARSE_ERROR }
            ),
    })

    const { parsedData: certData } = useCaData({
        data: cert !== '' ? [cert] : [],
        onError: (e) =>
            Notification.error(
                { title: _(t.certificationParsingError), message: e },
                { notificationId: notificationId.HUB_DPS_LINKED_HUBS_ADD_PAGE_CERT_PARSE_ERROR }
            ),
    })

    const [caModalData, setCaModalData] = useState<{ title: string; subTitle: string; data?: {}[] | string; dataChain: any }>({
        title: _(t.caPoolDetail),
        subTitle: '',
        data: undefined,
        dataChain: undefined,
    })

    const handleDeleteCaItem = useCallback(
        (id: number) => {
            const newData = cloneDeep(caPool)
            newData.splice(id, 1)
            setValue(`${prefix}tls.caPool`, newData, { shouldDirty: true, shouldTouch: true })
        },
        [caPool, prefix, setValue]
    )

    const handleViewCa = useCallback(
        (id: number) => {
            const caItem = caPoolData.find((item: { id: number; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.caPoolDetail), subTitle: caItem.name, data: caItem.data || caItem.name, dataChain: caItem.dataChain })
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [caPoolData]
    )

    const handleViewCert = useCallback(
        (id: number) => {
            const certItem = certData.find((item: { id: number; name: string; data: {}[] }) => item.id === id)
            setCaModalData({ title: _(t.caPoolDetail), subTitle: certItem.name, data: certItem.data || certItem.name, dataChain: certItem.dataChain })
        },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [certData]
    )

    return (
        <div>
            <Spacer type='pt-12'>
                <h2 css={commonStyles.subHeadline}>TLS</h2>
            </Spacer>

            <p css={commonStyles.description}>Short description...</p>

            <h3 css={commonStyles.groupHeadline}>{_(t.caPool)}</h3>

            <Spacer type='mb-5'>
                <Controller
                    control={control}
                    name='certificateAuthority.grpc.tls.useSystemCaPool'
                    render={({ field: { onChange, value } }) => (
                        <TileToggle
                            checked={(value as boolean) ?? false}
                            name={_(t.useSystemCAPool)}
                            onChange={(e) => {
                                updateField('certificateAuthority.grpc.keepAlive.permitWithoutStream', e.target.value === 'on')
                                onChange(e)
                            }}
                        />
                    )}
                />
            </Spacer>

            <Dropzone
                smallPadding
                customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                description={_(t.uploadCertDescription)}
                maxFiles={10}
                onFilesDrop={(files) => {
                    setTimeout(() => {
                        setValue(`${prefix}tls.caPool`, [...caPool, ...files.map((f) => pemToString(f))], {
                            shouldDirty: true,
                            shouldTouch: true,
                        })
                    }, 100)
                }}
                renderThumbs={false}
                title={_(t.uploadCertTitle)}
                validator={(file) => nameLengthValidator(file)}
            />

            {caPoolData && caPoolData.length > 0 && (
                <Spacer type='pt-6'>
                    <CaList
                        actions={{
                            onDelete: handleDeleteCaItem,
                            onView: handleViewCa,
                        }}
                        data={caPoolData}
                        i18n={{
                            download: _(g.download),
                            delete: _(g.delete),
                            view: _(g.view),
                        }}
                    />
                </Spacer>
            )}

            <h3 css={commonStyles.groupHeadline}>{_(t.privateKey)}</h3>

            {key === '' ? (
                <Dropzone
                    smallPadding
                    customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                    description={_(t.uploadCertKeyDescription)}
                    maxFiles={1}
                    onFilesDrop={(files) => {
                        setTimeout(() => {
                            setValue(`${prefix}tls.key`, pemToString(files[0]), {
                                shouldDirty: true,
                                shouldTouch: true,
                            })
                        }, 100)
                    }}
                    renderThumbs={false}
                    title={_(t.uploadCertTitle)}
                    validator={(file) => nameLengthValidator(file, true)}
                />
            ) : (
                <Spacer type='pt-2'>
                    <CaList
                        actions={{
                            onDelete: () => setValue(`${prefix}tls.key`, '', { shouldDirty: true, shouldTouch: true }),
                        }}
                        data={[{ id: 0, name: key, data: [], dataChain: '' }]}
                        i18n={{
                            download: _(g.download),
                            delete: _(g.delete),
                            view: _(g.view),
                        }}
                    />
                </Spacer>
            )}

            <h3 css={commonStyles.groupHeadline}>{_(t.certificate)}</h3>

            {cert === '' ? (
                <Dropzone
                    smallPadding
                    customFileRenders={[{ format: 'pem', icon: 'icon-file-pem' }]}
                    description={_(t.uploadCertDescription)}
                    maxFiles={1}
                    onFilesDrop={(files) => {
                        setTimeout(() => {
                            setValue(`${prefix}tls.cert`, pemToString(files[0]), {
                                shouldDirty: true,
                                shouldTouch: true,
                            })
                        }, 100)
                    }}
                    renderThumbs={false}
                    title={_(t.uploadCertTitle)}
                    validator={(file) => nameLengthValidator(file)}
                />
            ) : (
                <Spacer type='pt-2'>
                    <Loadable condition={certData.length > 0}>
                        <CaList
                            actions={{
                                onDelete: () => setValue(`${prefix}tls.cert`, '', { shouldDirty: true, shouldTouch: true }),
                                onView: handleViewCert,
                            }}
                            data={certData}
                            i18n={{
                                download: _(g.download),
                                delete: _(g.delete),
                                view: _(g.view),
                            }}
                        />
                    </Loadable>
                </Spacer>
            )}

            <CaPoolModal
                data={caModalData?.data}
                dataChain={caModalData?.dataChain}
                i18n={i18n}
                onClose={() => setCaModalData({ title: '', subTitle: '', data: undefined, dataChain: undefined })}
                show={caModalData?.data !== undefined}
                subTitle={caModalData.subTitle}
                title={caModalData.title}
            />
        </div>
    )
}

SubStepTls.displayName = 'SubStepTls'

export default SubStepTls
