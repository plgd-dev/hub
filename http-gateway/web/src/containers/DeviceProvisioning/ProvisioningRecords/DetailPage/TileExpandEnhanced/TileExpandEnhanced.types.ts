import { Props as TileExpandEnhancedProps } from '@shared-ui/components/Atomic/TileExpand/TileExpand.types'

export type TileExpandEnhancedType = {
    data: {
        status: {
            coapCode: number
            errorMessage?: string
            date?: string
        }
    }
    noSpace?: boolean
    divWrapper?: boolean
} & Pick<TileExpandEnhancedProps, 'information' | 'title'>
