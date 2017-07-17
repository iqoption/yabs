import _ from 'lodash';
import docViewsRegistry from 'ui/registry/doc_views';

import crashHtml from './crash.html';

docViewsRegistry.register(function () {
    return {
        title: 'Crash',
        order: 9,
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