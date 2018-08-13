// const api = require('./src/lib/api');
const Router = require('./src/lib/simple-router');

const router = new Router({
  middleware: [
    (event) => {
      console.log(event);
      return event;
    }
  ],
  routes: [
    [
      'GET', '/public-ip', (event) => {
        const clientIp = event.headers['X-Forwarded-For'] || 'unknown';
        return {
          statusCode: 200,
          body: JSON.stringify({
            publicIp: clientIp
          })
        }
      }
    ]
  ],
  defaultRoute: (event) => ({
    statusCode: 404,
    body: JSON.stringify({
      message: 'Resource not found',
      resource: event.path
    })
  })
});

exports.handler = async (event, context) => {
  event._context = context;
  try {
    return await router.route(event);
  } catch (e) {
    console.error(e);
    return {
      statusCode: 501,
      body: {
        message: e.message
      }
    };
  }
};

