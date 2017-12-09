/* eslint no-var: 0 */

var path = require("path");
var webpack = require('webpack');

function createConfig(isProductionMode) {
  var config = {
    plugins: [ 
      new webpack.DefinePlugin({
        'process.env': {
          'NODE_ENV': isProductionMode ? '"production"' : 'undefined'
        }
      })
      // new webpack.optimize.DedupePlugin() 
    ],
    module: {
      rules: [ 
        { test: /\.jsx?$/, exclude: /node_modules/, use: ['babel-loader']}
      ]
    },
    devServer: {
      historyApiFallback: {
        rewrites: [
          { from: /.*/, to: '/index.html' }
        ]
      },
      proxy: {
        '*': {
          target: 'http://localhost:8080',
          secure: false,
          headers: { "X-DevServer": "1" }
        }
      }
    }
  };

  if (isProductionMode) {
    config.plugins.push(new webpack.optimize.UglifyJsPlugin({
      minimize: true,
      compress: {
        warnings: false
      },
      sourceMap: false
    }));
    config.devtool = 'hidden-source-map';
  }
  else {
    config.devtool = 'cheap-module-eval-source-map';
  }
  return config;
}

module.exports = [
  // Components library. 
  // Debugging info is useless, so forcing production.
  Object.assign({}, createConfig(true), {
   entry: { server: './js/server.js' },
   output: {
     libraryTarget: "umd",
     library: "server",
     path: path.join(__dirname, "server"),
     filename: "[name].js"
   },
   devtool: undefined
  }),
  
  // Client
  Object.assign({}, createConfig(process.env.NODE_ENV === 'production'), {
    entry: { app: './js/client.js' },
    output: {
      path: path.join(__dirname, "server", "public", "js"),
      publicPath: "/js",
      filename: "[name].js"
    }
  })
];
