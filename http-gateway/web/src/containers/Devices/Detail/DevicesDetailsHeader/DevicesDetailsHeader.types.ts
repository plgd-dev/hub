import { ResourcesType } from '@/containers/Devices/Devices.types'

export type Props = {
    deviceId: string
    deviceName: string
    deviceOnboardingResourceData: any
    incompleteOnboardingData: boolean
    isOwned: boolean
    isUnregistered: boolean
    onOwnChange: () => void
    onboardButtonCallback?: () => void
    onboardResourceLoading: boolean
    onboarding: boolean
    openDpsModal: () => void
    openOnboardingModal: () => void
    resources: ResourcesType[]
}
