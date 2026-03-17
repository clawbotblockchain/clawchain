"""SQLite models for the ClawChain Gateway worker registry."""

import datetime
import uuid

from sqlalchemy import Boolean, Column, DateTime, Integer, String, create_engine
from sqlalchemy.orm import DeclarativeBase, Session, sessionmaker


class Base(DeclarativeBase):
    pass


class Worker(Base):
    __tablename__ = "workers"

    id = Column(String, primary_key=True, default=lambda: str(uuid.uuid4()))
    name = Column(String, nullable=False)
    platform = Column(String, default="unknown")
    key_name = Column(String, nullable=False, unique=True)
    address = Column(String, nullable=False, unique=True)
    ping_token = Column(String, nullable=False, default=lambda: str(uuid.uuid4()))
    active = Column(Boolean, default=True)
    last_ping = Column(DateTime, nullable=True)
    heartbeats_sent = Column(Integer, default=0)
    created_at = Column(DateTime, default=datetime.datetime.utcnow)


def get_engine(database_url: str = "sqlite:///./data/gateway.db"):
    return create_engine(database_url, connect_args={"check_same_thread": False})


def get_session_factory(engine) -> sessionmaker:
    return sessionmaker(bind=engine, class_=Session)


def init_db(engine):
    Base.metadata.create_all(engine)
