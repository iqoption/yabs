import {resolve} from 'path';

export default function (kibana) {
    return new kibana.Plugin({
        require: ['elasticsearch'],

        uiExports: {
            docViews: [
                'plugins/gl_crash_plugin/gl_crash_views'
            ]
        }
    });
};