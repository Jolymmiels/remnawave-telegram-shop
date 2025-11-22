import loadingOverlay from './loading-overlay'
import ringProgress from './ring-progress'
import notification from './notification'
import breadcrumbs from './breadcrumbs'
import buttons from './buttons'
import layouts from './layouts'
import tooltip from './tooltip'
import charts from './charts'
import drawer from './drawer'
import inputs from './inputs'
import badge from './badge'
import table from './table'
import card from './card'
import menu from './menu'

export default {
    ...card,
    ...badge,
    ...breadcrumbs,
    ...buttons,
    ...charts,
    ...drawer,
    ...inputs,
    ...loadingOverlay,
    ...menu,
    ...notification,
    ...ringProgress,
    ...table,
    ...tooltip,
    ...layouts
}
