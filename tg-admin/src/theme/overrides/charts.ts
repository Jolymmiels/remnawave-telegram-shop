import { BarChart } from '@mantine/charts'

export default {
    BarChart: BarChart.extend({
        defaultProps: {
            barProps: {
                radius: 8
            }
        }
    })
}
