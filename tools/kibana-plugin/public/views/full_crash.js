import _ from 'lodash';
import docViewsRegistry from 'ui/registry/doc_views';

import crashHtml from './full_crash.html';

docViewsRegistry.register(function () {
    return {
        title: 'Full Crash',
        order: 10,
        directive: {
            template: crashHtml,
            scope: {
                hit: '=',
                indexPattern: '=',
                filter: '=',
                columns: '=',
            },
            controller: function ($scope) {
                $scope.mapping = $scope.indexPattern.fields.byName;
                $scope.flattened = $scope.indexPattern.flattenHit($scope.hit);
                $scope.formatted = $scope.indexPattern.formatHit($scope.hit);
                $scope.fields = _.keys($scope.flattened).sort();
            }
        }
    };
});