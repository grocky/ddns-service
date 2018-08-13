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
    'GET', '/public-ip', (event) => {
      const clientIp = event.headers['X-Forwarded-For'] || 'unknown';
      return {
        statusCode: 200,
        body: {
          publicIp: clientIp
        }
      }
    }
  ],
  defaultRoute: (event) => ({
    statusCode: 404,
    body: {
      message: 'Resource not found',
      resource: event.path
    }
  })
});

exports.handler = async (event, context) => {
  event._context = context;
  try {
    return await router.route(event);
  } catch (e) {
    console.error(e);
    return {
      statusCode: 500,
      body: e.message
    };
  }
};

