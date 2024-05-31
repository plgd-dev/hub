import { Inputs } from '../EnrollmentGroups.types'

export type Props = {
    defaultFormData: {
        id: string
        hubsData: {
            authorization: any
            certificateAuthority: any
            coapGateway: string
            id: string
            name: string
        }[]
    } & Inputs
    resetIndex?: number
}
