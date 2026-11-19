import { PrismaClient } from '@prisma/client';
import { PrismaPg } from '@prisma/adapter-pg';
import { config } from '../config.js';

const adapter = new PrismaPg({ connectionString: config.databaseUrl });

const db = new PrismaClient({ adapter });

export default db;
